import datetime
import time
import pytz
import elasticsearch
import json
import sys
import requests

from arango import ArangoClient
from elasticsearch import helpers
from pyArango.connection import Connection
from pyArango.database import Database

class ArangoDB:
    def __init__(self, url, db, username, passwd):
        self.db = Connection(arangoURL=url).databases[db]
    
    def fetch_data(self, aql):
        return self.db.AQLQuery(aql, rawResults=True)
    
    def has_collection(self, collection):
        return self.db.hasCollection(collection)
    
    def add_index(self, collection, indexList):
        return self.db[collection].ensureHashIndex(indexList)

class ElasticSearch:
    def __init__(self, url):
        self.es= elasticsearch.Elasticsearch(url, request_timeout=100)
    
    def bulk_to_es(self, dlist):
        actions = []
        count = 0
        
        for action in dlist:
            actions.append(action)
            count += 1
            
            if len(actions) == 500:
                helpers.bulk(self.es, actions)
                del actions[0:len(actions)]
        
        if len(actions) > 0:
            helpers.bulk(self.es, actions)
            del actions[0:len(actions)]
            
        return count
    
    def get_cause_nodes_from_es(self):
        max_timestamp = int(time.time()) * 1000
        min_timestamp = max_timestamp - 3 * 60 * 1000
        print(min_timestamp, max_timestamp)
        
        query = {
                    "bool": {
                        "must": [
                            {"range": {"Timestamp": {"gte": min_timestamp,"lte": max_timestamp}}}
                        ]
                    }
                }
        resp = self.es.search(index="gala_cause_inference-*", query=query)
        hits= resp['hits']['hits']
        timestamp = sys.maxsize
        nodes = {}
        if len(hits) >= 1:
            print('hit len {}'.format(len(hits)))
            for hit in hits:
                cause_metrics = hit['_source']['Resource']['cause_metrics']
                for cause_metric in cause_metrics:
                    paths = cause_metric['path']
                    if hit['_source']['Timestamp'] < timestamp:
                        timestamp = hit['_source']['Timestamp']
                        nodes = {}
                    elif hit['_source']['Timestamp'] == timestamp:
                        # one timestamp multi records
                        timestamp = hit['_source']['Timestamp']
                    else:
                        continue
                    for path in paths:
                        if path['entity_id'] not in nodes:
                            nodes[path['entity_id']] = {'metric_id': path['metric_id'], 'desc': path['desc'], 'count': 1}
                        else:
                            if nodes[path['entity_id']]['desc'] is None or path['desc'] in nodes[path['entity_id']]['desc']:
                                continue
                            else:
                                nodes[path['entity_id']]['metric_id'] += "," + path['metric_id']
                                if nodes[path['entity_id']]['count'] == 1:
                                    temp = nodes[path['entity_id']]['desc']
                                    nodes[path['entity_id']]['desc']  += "1." + temp
                                nodes[path['entity_id']]['count'] += 1
                                nodes[path['entity_id']]['desc'] += '\n' + str(nodes[path['entity_id']]['count'] ) + '.' + path['desc']
                        print(path['entity_id'], nodes[path['entity_id']])
                    print("-------------------")
        print(timestamp)
        return {'nodes': nodes, 'timestamp': timestamp}

    def has_record_in_graph(self, ts):
        query = {
                    "bool": {
                        "must": [
                            {"range": {"ts": {"gte": ts, "lte": ts}}}
                        ]
                    }
                }
        resp = self.es.search(index="aops_graph2", query=query)
        hits = resp['hits']['hits']
        if len(hits) >= 1:
            return True
        else:
            return False
        
class AOps:
    def __init__(self):
        self.arangodbUrl = 'http://localhost:8529'
        self.esUrl = 'http://localhost:9200'
        self.promethusUrl = 'http://localhost:9090'
        self.db_client = ArangoDB(self.arangodbUrl, 'spider', 'root', '')
        self.es_client = ElasticSearch(self.esUrl)
        self.edge_collection = ['belongs_to', 'connect', 'has_vhost', 'is_peer', 'runs_on', 'store_in']
        self.bad_nodes = {}
        self.hosts_map = {}
    
    def getHostMapFromPromethus(self):
        url = self.promethusUrl + "/api/v1/query?query=gala_gopher_host_value"
        rsp = requests.get(url).json()
        if 'status' in rsp and rsp['status'] == 'success':
            for i in rsp['data']['result']:
                self.hosts_map[i['metric']['machine_id']] = i['metric']['job']
    
    def get_timestamp(self, ts_sec):
        if ts_sec == 0:
            cur_ts_sec = int(time.time())
        else:
            cur_ts_sec = ts_sec
        
        aql = "For t in Timestamps FILTER TO_NUMBER(t._key) <= {} SORT t._key DESC LIMIT 1 return t._key".format(cur_ts_sec)
        timestamp = self.db_client.fetch_data(aql)
        if len(timestamp) != 0:
            return timestamp[0]
        else:
            return 0
    
    def get_metrics_str_from_node(self, node):
        if node['type'] == 'host':
            node['metrics'].pop('value') # host metrics-value no use
        
        # set bad nodes
        node_id = node['_id'].split('/')[1]
        if node_id in self.bad_nodes:
            node['metrics']['health_status'] = 'False'
            node['metrics']['health_desc'] = self.bad_nodes[node_id]['desc']
            node['metrics']['health_metric'] = self.bad_nodes[node_id]['metric_id']
        else:
            node['metrics']['health_status'] = 'True'
            node['metrics']['health_desc'] = ''
            node['metrics']['health_metric'] = ''

        # set chao proc node to bad status
        if 'comm' in node and 'chaos_os'  in node['comm']:
            node['metrics']['health_status'] = 'False'
            node['metrics']['health_desc'] = ''
            node['metrics']['health_metric'] = ''

        return json.dumps(node['metrics'])

    def get_node_info(self, edge_origion, node, dic):
        node_type= node['type']
        dic['_source'][edge_origion + '_type'] = node_type
        dic['_source'][edge_origion + '_level'] = node['level']
        
        if node_type == 'proc':
            if 'comm' in node:
                dic['_source'][edge_origion + '_comm']  = node['comm'] + node['_key'][len(node['machine_id']):]
            else:
                dic['_source'][edge_origion + '_comm']  = node['_key'][len(node['machine_id']):]
        elif node_type == 'host':
            if node['machine_id']  in self.hosts_map.keys():
                dic['_source'][edge_origion + '_comm'] = self.hosts_map[node['machine_id']]
            elif 'ceph' in node['hostname']:
                dic['_source'][edge_origion + '_comm'] = 'ceph_host_'  + node['ip_addr']
            else:
                dic['_source'][edge_origion + '_comm']  = node['host_type'] + '_host_' + node['ip_addr']
        elif node_type == 'thread':
            dic['_source'][edge_origion + '_comm'] = node['comm'] + node['_key'][len(node['machine_id']):]
        elif node_type == 'block':
            dic['_source'][edge_origion + '_comm'] = node['disk_name'] + '_' + node['blk_name'] \
                + node['_key'][len(node['machine_id']):]
        else:
            dic['_source'][edge_origion + '_comm'] = node['_key'][len(node['machine_id']) + 1:]
        
        
        # set bad nodes
        node_id = node['_id'].split('/')[1]
        if node_id in self.bad_nodes:
            dic['_source'][edge_origion + '_status'] = 'bad'
        else:
            dic['_source'][edge_origion + '_status'] = 'good'

        # set chao proc node to bad status
        if 'chaos_os' in dic['_source'][edge_origion + '_comm']:
            dic['_source'][edge_origion + '_status'] = 'bad'

    def get_node_by_from(self, edge_from, node, dic):
        if node['_id'] == edge_from:
            self.get_node_info('src', node, dic)
            dic['_source']['metric'] = self.get_metrics_str_from_node(node)
        else:
            self.get_node_info('dst', node, dic)
            dic['_source']['dst_metric'] = self.get_metrics_str_from_node(node)
    
    # filter redis/gaussdb proc, add proc_comm
    def filter_proc(self, data):
        redis_container_list = []
        gaussdb_container_list = []
        redis_tcp_list = []
        gaussdb_tcp_list = []
        for it in data:
            if 'redis' in it['_source']['src_comm']:
                it['_source']['proc_comm'] = 'redis'
                redis_container_list.append(it['_source']['dst'])
                continue
            if 'gaussdb' in it['_source']['src_comm']:
                it['_source']['proc_comm'] = 'gaussdb'
                gaussdb_container_list.append(it['_source']['dst'])
                continue
            if 'redis' in it['_source']['dst_comm']:
                redis_tcp_list.append(it['_source']['src'])
                continue
            if 'gaussdb' in it['_source']['dst_comm']:
                gaussdb_tcp_list.append(it['_source']['src'])
                continue
            

        for it in data:
            if it['_source']['src'] in redis_container_list or it['_source']['src'] in redis_tcp_list:
                it['_source']['proc_comm'] = 'redis'
                continue
            if it['_source']['src'] in gaussdb_container_list or it['_source']['src'] in gaussdb_tcp_list:
                it['_source']['proc_comm'] = 'gaussdb'
                continue
    
    def fetch_graph_data(self):
        self.getHostMapFromPromethus()
        cause_data = self.es_client.get_cause_nodes_from_es()
        if cause_data['timestamp'] == sys.maxsize:
            ag_ts = self.get_timestamp(0);
            ts = int(ag_ts) * 1000
        else:
            ag_ts = self.get_timestamp(int(cause_data['timestamp'] / 1000))
            ts = cause_data['timestamp']
        self.bad_nodes = cause_data['nodes']
        print('ts:{} ag_ts:{}'.format(ts, ag_ts))
        results = []
        if self.es_client.has_record_in_graph(ts):
            print('{} has recorded'.format(ts))
            return
        for edge_collection in self.edge_collection:
            if not self.db_client.has_collection(edge_collection):
                continue
            
            aql = "For doc in " + edge_collection + \
                    " FILTER " \
                    " doc.timestamp ==" +  ag_ts + \
                    "RETURN doc"
            
            edges = self.db_client.fetch_data(aql)
            
            for edge in edges:
                edge_from = edge['_from']
                edge_to = edge['_to']
                edge_type = edge['type']
                edge_layer = edge['layer']
                
                dic = {
                    "_index": "aops_graph2",
                    "_source": {
                        "ts": ts,
                        "timestamp": datetime.datetime.fromtimestamp(ts / 1000, pytz.utc),
                        "edge_type": edge_type,
                        "edge_layer": edge_layer,
                        "src": edge_from,
                        "dst": edge_to
                    }
                }
                
                aql = "For doc in ObserveEntities_" + ag_ts + \
                        " Filter doc._id == '" + edge['_from'] + "' ||" + \
                        " doc._id == '" + edge['_to'] + "' LIMIT 10 return doc"
                nodes = self.db_client.fetch_data(aql)
                for node in nodes:
                    self.get_node_by_from(edge['_from'], node, dic)
                results.append(dic)
        print("node length", len(results))

        self.filter_proc(results)

        count = self.es_client.bulk_to_es(results)
        print("write to es count is:", count)
    
    def set_graph_timestamp_index(self):
        for edge_collection in self.edge_collection:
            if not self.db_client.has_collection(edge_collection):
                continue
            self.db_client.add_index(edge_collection, ['timestamp'])    
    
if __name__ == "__main__":
    AOps().set_graph_timestamp_index();
    while True:
        start_time = int(time.time())
        AOps().fetch_graph_data()
        end_time = int(time.time())
        print("fetch graph data cost time:{}".format(end_time - start_time))
        time.sleep(10)
        print('-----------------------------------------------')
            
            
        
        
