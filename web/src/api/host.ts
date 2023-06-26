import request from './request'

// 获取监控主机ip
export function getMacIp() {
  return request({
    url: '/monitor/info',
    method: 'get',
  })
}

// 获取全部主机列表
export function getIpList() {
  return request({
    url: '/host/lists',
    method: 'get',
  })
}

// 已安装监控组件的主机列表分页
export function getExporterList(data: object) {
  return request({
    url: '/hosts/lists',
    method: 'get',
    params: data
  })
}

// 已安装监控组件的主机列表全部
export function getAllExporterList() {
  return request({
    url: '/hosts/all',
    method: 'get',
  })
}

// 模糊查询主机列表
export function searchIpList(data: object) {
  return request({
    url: '/hosts/search',
    method: 'post',
    data
  })
}

// 未安装监控组件的主机列表全部
export function getAllNoExporterList(data: object) {
  return request({
    url: '/hosts/notInstalled',
    method: 'get',
    params: data
  })
}

//  监控组件注册
export function agentRegister(data: object) {
  return request({
    url: '/agent/register',
    method: 'post',
    data
  })
}

//  监控组件升级
export function agentUpgrade(data: object) {
  return request({
    url: '/agent/update',
    method: 'post',
    data
  })
}

//  监控组件卸载
export function agentUninstall(data: object) {
  return request({
    url: '/agent/remove',
    method: 'post',
    data
  })
}