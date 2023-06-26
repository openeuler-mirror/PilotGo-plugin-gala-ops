<template>
  <div class="install">
    <pm-table :getData="getAllNoExporterList" ref="installRef" @handleSelect="handleSelect">
      <el-table-column type="selection" width="50" />
      <el-table-column fixed="left" prop="ip" label="IP地址" width="140">
      </el-table-column>
      <el-table-column prop="departmentName" label="部门" width="220" />
      <el-table-column prop="operatingSystem" label="操作系统信息" width="280" />
      <el-table-column prop="version" label="版本" />
      <el-table-column prop="architecture" label="CPU架构" width="220" />
    </pm-table>
    <el-button class="el-button2" :disabled="selectedIds.length == 0" @click="handleInstall">注册</el-button>
  </div>
</template>

<script setup lang="ts">
import { getAllNoExporterList, agentRegister } from '@/api/host';
import { ref } from 'vue';
import pmTable from "@/components/PmTable.vue";
import { ElMessage } from 'element-plus';
import Cookies from 'js-cookie'
const installRef = ref(null)
const selectedIds = ref([])
const emit = defineEmits(['handleCancle'])
const handleSelect = (ids: any) => {
  selectedIds.value = ids;
}
// 注册
const handleInstall = () => {
  agentRegister({ hostIds: selectedIds.value, AuthToken: Cookies.get('Admin-Token'), webIp: window.location.hostname }).then(res => {
    if (res.data && res.data.code === 200) {
      ElMessage.success(res.data.msg);
      emit('handleCancle', 'success')
    } else {
      ElMessage.error('注册失败，请重试')
      emit('handleCancle')
    }
  })
}


</script>

<style scoped lang="scss">
.install {
  width: 100%;
  max-height: 600px;
  overflow-y: scroll;
}
</style>