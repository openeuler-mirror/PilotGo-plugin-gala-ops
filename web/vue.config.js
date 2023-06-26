module.exports = {
  devServer: {
    headers: {
      'Access-Control-Allow-Origin': '*',
    },
  //   proxy: {
  //     '/api': {
  //       target: 'http://172.30.23.106:8088',
  //       ws: false,
  //       changeOrigin: true,
  //       pathRewrite: {
  //         '^api/': ''
  //       }
  //     }
  //   },
  //   port: 8080,
  //   host: '0.0.0.0',
  },

  // 静态资源
  publicPath: "./",
  outputDir: "dist",
  assetsDir: "static/",
  indexPath: "index.html",
  filenameHashing: true,

  chainWebpack: (config) => {

    config.resolve.symlinks(true);
  },
  configureWebpack: {
    output: {
      //资源打包路径
      library: 'vueApp',
      libraryTarget: 'umd'
    }
  }
}
