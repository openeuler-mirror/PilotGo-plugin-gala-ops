/* eslint-disable */
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// fairy自定义
declare module '*.scss';
declare module '*.md';
declare module 'vue3-infinite-list';
declare module 'vue-search-highlight';
declare module 'vue-grid-layout';
declare module 'marked';
declare module '@kangc/v-md-editor/lib/preview'
declare module '@kangc/v-md-editor/lib/theme/vuepress.js'
declare module 'element-plus/dist/locale/zh-cn.mjs'
declare module 'js-cookie'
declare module 'crypto-js/enc-base64'
declare module 'crypto-js/enc-utf8'
declare interface Window {
  microApp: any
}
