import axios from 'axios'
import router from '@/router'
import Cookies from 'js-cookie'


// 1.创建axios实例
const request = axios.create({
  baseURL: '/api/ismp-plugin-monitor/',
  timeout: 5000,
  headers: {
    // 设置后端需要的传参类型
    'Content-Type': 'application/json',
    'token': '',
    'X-Requested-With': 'XMLHttpRequest',
  },
})

// 2.1添加请求拦截器
request.interceptors.request.use(config => {
  if (Cookies.get('Admin-Token')) {
    config.headers!['authToken'] = Cookies.get('Admin-Token');
  }
  let url = config.url
  // get参数编码 为了prometheus的请求url不被转义
  if (config.method === 'get' && config.params) {
    url += '?'
    let keys = Object.keys(config.params)
    for (let key of keys) {
      url += `${key}=${encodeURIComponent(config.params[key])}&`
    }
    url = url!.substring(0, url!.length - 1)
    config.params = {}
  }
  config.url = url;
  return config
}, error => {
  return Promise.reject(error);
});

// 2.2添加响应拦截器
request.interceptors.response.use((response: any) => {
  if (response.data && response.data.code == '401') {
    router.push("/")
    if (window.__MICRO_APP_ENVIRONMENT__) {
      window.microApp.setGlobalData({ code: '401' })
    }
  } else {
    return response;
  }
  // return response;
}, error => {
  if (error.response) {
    switch (error.response.status) {
      case 401:
        router.push("/home")
        if (window.__MICRO_APP_ENVIRONMENT__) {
          window.microApp.setGlobalData({ code: '401' })
        }
    }
    return Promise.reject(error.response.data);
  }
});

export default request
