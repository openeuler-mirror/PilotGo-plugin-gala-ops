<template>
  <div class="top">
    <span class="top-title">监控主机管理</span>
  </div>
  <el-dialog v-model="dialog" v-if="type === 'install'" :width="width" :title="title" left>
    <install-form @handleCancle="handleClose" />
  </el-dialog>
  <el-dialog v-else-if="type === 'upgrade' || 'uninstall'" v-model="dialog" :width="width" :title="title"
    :show-close="false">
    <template #header="{ close, titleId, titleClass }">
      <div class="top-dialog-header">
        <el-icon color="red" size="22">
          <WarningFilled />
        </el-icon>
        <span class="top-dialog-header-title" :id="titleId" :class="titleClass">{{
          '确定' + typeText + checkedIps[0] + '等'
          + checkedIps.length + '(台)主机的监控组件吗？'
        }}</span>
      </div>
    </template>
    <upgrade-form :idList="checkedIds" @handleCancle="handleClose" :hintText="hintText" :typeText="typeText" />
  </el-dialog>
  <div class="list">
    <div class="operation">
      <div class="operation-select">
        <el-dropdown split-button type="primary" class="operation-select-checkbox" @command="handleCheckChange"
          @click.stop="handleCheckClick($event, !checkedState)">
          <el-checkbox v-model="checkedState" @change.native="handleCheckState" />
          <span class="operation-select-spanText" v-show="checkedState && checkedCount > 0">选择个数（{{ checkedCount
          }}个）</span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="不选择">不选择（0个）</el-dropdown-item>
              <el-dropdown-item command="选择当前页">{{ '选择当前页（' + currentNum + '个）' }}</el-dropdown-item>
              <el-dropdown-item command="选择所有">{{ '选择所有（' + page.total + '个）' }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <div class="operation-select-input">
          <el-autocomplete v-model="macIp" :fetch-suggestions="querySearch" popper-class="my-autocomplete"
            placeholder="请输入IP" @select="handleSelectIP" @change="handleInputIP">
            <template #prefix>
              <el-icon class="el-input__icon" @click="handleSearchClick">
                <Search />
              </el-icon>
            </template>
            <template #suffix>
              <el-icon class="el-input__icon" @click="handleArrowClick">
                <ArrowDown />
              </el-icon>
            </template>
            <template #default="{ item }">
              <div class="value">{{ item.ip }}</div>
            </template>
          </el-autocomplete>
        </div>
      </div>
      <div class="operation-btn">
        <el-button plain class="el-button1" type="primary" @click="handleRegister">监控注册</el-button>
        <el-button plain class="el-button2" type="primary" @click="handleUpgrade"
          :disabled="checkedIps.length == 0">监控升级</el-button>
        <el-button plain class="el-button2" type="primary" @click="handleUninstall"
          :disabled="checkedIps.length == 0">监控卸载</el-button>
      </div>
    </div>
    <el-table ref="multipleTableRef" :data="tableData" class="table" @selection-change="handleSelectionChange"
      v-loading="loading">
      <el-table-column type="selection" width="50" />
      <el-table-column fixed="left" label="IP地址" width="120" prop="ip">
        <template #default="scope">
          <el-button link type="primary" size="small" @click="handleIpJump(scope.row.ip)">{{ scope.row.ip }}</el-button>
        </template>
      </el-table-column>
      <el-table-column prop="deptName" label="部门" width="220" />
      <el-table-column prop="operatingSystem" label="操作系统信息" />
      <el-table-column prop="version" label="版本" width="220" />
      <el-table-column prop="architecture" label="CPU架构" width="220" />
      <el-table-column prop="monitorVersion" label="gala-gopher版本" width="260" />
      <el-table-column prop="agentStatus" label="agent状态" width="160">
        <template #default="scope">
          <span style="display:flex;align-items: center; justify-content: center;">
            <el-icon v-if="scope.row.agentStatus === '连接'" color="#67c23a">
              <SuccessFilled />
            </el-icon>
            <el-icon v-else color="#ff0000">
              <CircleCloseFilled />
            </el-icon>
            <span style="margin:0 4px;">{{ scope.row.agentStatus }}</span>
          </span>
        </template>
      </el-table-column>
      <el-table-column prop='registTime' sortable label="监控组件注册时间" />
    </el-table>
    <div class="pagination">
      <el-config-provider :locale="zhCn">
        <el-pagination v-model:current-page="page.currentPage" v-model:page-size="page.pageSize"
          :page-sizes="[20, 25, 50, 75, 100]" :small="page.small" :disabled="page.disabled" :background="page.background"
          layout="total, sizes,prev, pager, next, jumper" :total="page.total" @size-change="getHostList"
          @current-change="getHostList" /></el-config-provider>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, watch } from 'vue';
import { ElTable } from 'element-plus'
import router from '@/router';
import { Search, ArrowDown } from '@element-plus/icons-vue';
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import { getExporterList, getAllExporterList, searchIpList } from '@/api/host';
import installForm from './hostForm/installForm.vue'
import upgradeForm from './hostForm/upgradeForm.vue'
import { useMacStore } from '@/store/mac';

onMounted(() => {
  getHostList();
  getAllExporterList().then(res => {
    if (res.data && res.data.code === 200) {
      ips.value = res.data.data
    }
  })
})
const page = reactive({
  total: 0,
  currentPage: 1,
  pageSize: 20,
  small: false,
  disabled: false,
  background: true,
})

// ip列表
interface IpItem {
  ip: string
}
const dialog = ref(false)
const type = ref('')
const title = ref('')
const width = ref('60%')
const typeText = ref('')
const hintText = ref('')
const loading = ref(false);
const tableData = ref([] as any);
const currentNum = ref(0); //复选框当前页数量
const checkedCount = ref(0)
const checkedState = ref(false)
const checkedIps = ref([] as string[])
const checkedIds = ref([] as Number[])
const macIp = ref('')
const ips = ref<IpItem[]>([])

const multipleTableRef = ref<InstanceType<typeof ElTable>>()

const querySearch = (queryString: string, cb: Function) => {
  const results = queryString
    ? ips.value.filter(createFilter(queryString))
    : ips.value
  // call callback function to return suggestion objects
  cb(results)
}
const createFilter = (queryString: any) => {
  return (restaurant: any) => {
    return (
      restaurant.ip.toLowerCase().indexOf(queryString.toLowerCase()) === 0
    )
  }
}
// 关闭dialog对话框
const handleClose = (closeType: string) => {
  dialog.value = false;
  type.value = '';
  title.value = '';
  hintText.value = '';
}

// 获取主机列表
const getHostList = () => {
  loading.value = false;
  /* getExporterList({ page: page.currentPage, size: page.pageSize }).then(res => {
    if (res.data && res.data.code === 200) {
      loading.value = false;
      tableData.value = res.data.data;
      currentNum.value = res.data.data.length;
      page.total = res.data.total;
    } else {
      loading.value = false;
      tableData.value = [];
      currentNum.value = 0;
      page.total = 0;
    }
  }) */
  let result = { "code": 200, "data": [{ "hostId": 1, "ip": "172.30.23.32", "deptName": "平台", "operatingSystem": "银河麒麟高级服务器操作系统", "version": "V10(SP1)", "architecture": "aarch64", "monitorVersion": "1.0", "registTime": "2023-04-18 14:20:22", "agentStatus": "连接" }], "ok": true, "page": 1, "size": 20, "total": 1 }
  tableData.value = result.data;
  currentNum.value = result.data.length;
  page.total = result.total;
}

// 输入回车事件 
const handleInputIP = (macIp: string) => {
  handleSelectIP({ ip: macIp })
}
// 选中ip主机事件
const handleSelectIP = (item: IpItem) => {
  loading.value = true;
  macIp.value = item.ip;
  searchIpList({ ip: item.ip }).then(res => {
    if (res.data && res.data.code === 200) {
      loading.value = false;
      tableData.value = res.data.data;
      currentNum.value = res.data.data.length;
      page.total = res.data.total;
    } else {
      loading.value = false;
    }
  })
}

// 点击搜索图标事件
const handleSearchClick = (ev: Event) => {
  loading.value = true;
  tableData.value = [];
  currentNum.value = 0;
  page.total = 0;
  searchIpList({ ip: macIp.value }).then(res => {
    if (res.data && res.data.code === 200) {
      loading.value = false;
      tableData.value = res.data.data;
      currentNum.value = res.data.data.length;
      page.total = res.data.total;
    } else {
      loading.value = false;
    }
  })
}

// 点击下拉图标事件
const handleArrowClick = (ev: Event) => {
  console.log('arrow', ev)
}

// 点击ip跳转大屏事件
const handleIpJump = (ip: string) => {
  useMacStore().setMacIp(ip + ":9100");
  if (window.__MICRO_APP_ENVIRONMENT__) {
    window.microApp.dispatch({ type: 'router', path: '/', microName: 'monitor' })
  }
  router.push({ name: 'home', query: { macIp: ip } })
}

// 点击跳转终端事件
const handleTerminal = (ip: string) => {
  // dispatch只接受对象作为参数  告诉主应用去terminal
  if (window.__MICRO_APP_ENVIRONMENT__) {
    window.microApp.dispatch({ type: 'router', path: '/terminal', microName: 'monitor' })
  }
  router.push({ name: 'terminal', query: { macIp: ip } })
}
// 注册
const handleRegister = () => {
  dialog.value = true;
  type.value = 'install';
  title.value = '监控组件注册';
  width.value = '70%'
}
// 升级
const handleUpgrade = () => {
  dialog.value = true;
  type.value = 'upgrade';
  title.value = '';
  width.value = '30%';
  typeText.value = '升级';
  hintText.value = '升级组件版本的记录可在日志界面查看';
}
// 卸载
const handleUninstall = () => {
  dialog.value = true;
  type.value = 'uninstall';
  title.value = '监控组件卸载';
  width.value = '30%';
  typeText.value = '卸载';
  hintText.value = '卸载监控组件的主机将从监控主机管理列表中删除，仍可在系统主机列表中查看相关信息';
}

// 表格复选框
const handleSelectionChange = (val: any[]) => {
  // 处理选中的行
  checkedIps.value = [];
  checkedIds.value = [];
  if (val.length > 0) {
    checkedState.value = true;
    val.forEach(item => {
      checkedIps.value.push(item.ip);
      checkedIds.value.push(item.hostId);
    });
    checkedCount.value = checkedIds.value.length;
  } else {
    checkedState.value = false;
  }
}

// 点击全局选择框事件
const handleCheckChange = (val: string) => {
  checkedIps.value = [];
  switch (val) {
    case '不选择':
      checkedIps.value = [];
      checkedState.value = false;
      checkedCount.value = 0;
      toggleSelection();
      break;
    case '选择当前页':
      checkedState.value = true;
      tableData.value.forEach((item: any) => {
        checkedIps.value.push(item.ip);
      });
      checkedCount.value = currentNum.value;
      toggleSelection(tableData.value)
      break;


    default:
      // 选择所有
      checkedState.value = true;
      checkedCount.value = page.total;
      ips.value.forEach((item: any) => {
        checkedIps.value.push(item.ip);
      });
      multipleTableRef.value!.clearSelection()
      multipleTableRef.value!.toggleAllSelection()
      break;
  }
}

// checkbox选中状态事件
const handleCheckState = (state: boolean) => {
  state ? handleCheckChange('选择所有') : handleCheckChange('不选择');
}

// 下拉框左侧部分点击事件
const handleCheckClick = (event: any, val: any) => {
  if (['operation-select-spanText', 'el-button el-button--primary'].includes(event.target!.className)) {
    val ? handleCheckChange('选择所有') : handleCheckChange('不选择');
  }
}

// 处理复选框选中事件
const toggleSelection = (rows?: any[]) => {
  if (rows) {
    rows.forEach((row) => {
      // TODO: improvement typing when refactor table
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      multipleTableRef.value!.toggleRowSelection(row, true)
    })
  } else {
    multipleTableRef.value!.clearSelection()
  }
}


</script>


<style scoped lang="scss">
.top {
  width: 97.4%;
  margin: 0 auto;
  height: 64px;
  display: flex;
  justify-content: space-between;
  align-items: center;

  &-title {
    font-size: 20px;
    color: #222;
    font-weight: 500;
    display: inline-block;
  }

  &-dialog {
    &-header {
      text-align: left;
      display: flex;
      align-items: center;

      &-title {
        font-size: 14px;
        font-weight: bold;
        padding-left: 10px;
      }
    }
  }
}

.list {
  width: 98.4%;
  height: calc(100% - 64px - 20px);
  margin: 0 auto;
  background-color: #fff;
  padding: 0 20px;

  .operation {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    height: 84px;

    &-select {
      min-width: 454px;
      display: flex;
      align-items: center;

      &-spanText {
        font-size: 12px;
        padding-left: 4px;
      }

      &-input {
        margin-left: 10px;
      }

    }


    &-btn {
      width: 26%;
      display: flex;
      justify-content: flex-end;
    }
  }

  .table {
    width: 100%;
    height: calc(100% - 84px - 64px);
  }

  .pagination {
    width: 100%;
    height: 44px;
    display: flex;
    justify-content: flex-end;
  }
}
</style>