import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router'
import HostList from '../views/hostList.vue'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    redirect: '/host'
  },
  {
    path: '/host',
    name: 'host',
    component: HostList
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
