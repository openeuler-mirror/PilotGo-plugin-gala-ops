// entry
import './public-path'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { Router } from 'vue-router'
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import * as ElementPlusIconsVue from '@element-plus/icons-vue';
import './styles/main.scss';


declare global {
  interface Window {
    microApp: any
    __MICRO_APP_NAME__: string
    __MICRO_APP_ENVIRONMENT__: string
    __MICRO_APP_BASE_ROUTE__: string

  }
}

// 与基座进行数据交互
function handleMicroData(router: Router) {
  // 是否是微前端环境
  if (window.__MICRO_APP_ENVIRONMENT__) {
    // 监听基座下发的数据变化
    const data = window.microApp.getData();
    if (data) {
      dataRoute(data);
    }
    window.microApp.addDataListener((data: any) => {
      dataRoute(data)
    })

    function dataRoute(data: any) {
      // 当基座下发path时进行跳转
      if (data.path && data.path !== router.currentRoute.value.path) {
        data.ip && data.ip.length > 1 ? router.push({ name: data.path, query: { macIp: data.ip } }) : router.push(data.path as string)
      }
    }
  }
}


const app = createApp(App);

const pinia = createPinia();

app.use(pinia);
app.use(router);
app.use(ElementPlus)
// 引入element-plus图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}
app.mount('#app')

handleMicroData(router)

// 监听卸载操作
window.addEventListener('unmount', function () {
  app.unmount()
  console.log('微应用child-vue3卸载了')
})