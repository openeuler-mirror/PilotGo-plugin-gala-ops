<template>
  <div class="upContent">
    <span class="upContent-body">
      {{ hintText }}
    </span>
    <div class="upContent-footer">
      <el-button @click="handleCancle">取消</el-button>
      <el-button type="primary" @click="handleConfirm">
        确定
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { agentUpgrade, agentUninstall } from '@/api/host';
import Cookies from 'js-cookie';
import { ElMessage } from 'element-plus';
const props = defineProps({
  idList: {
    type: Array,
    default: []
  },
  hintText: {
    type: String,
    default: '',
  },
  typeText: {
    type: String,
    default: '升级'
  }
})
const emit = defineEmits(['handleCancle'])
const hintText = ref('');
onMounted(() => {
  hintText.value = props.hintText;
})
// 取消升级
const handleCancle = () => {
  emit('handleCancle')
}

// 确定
const handleConfirm = () => {
  let params = {
    hostIds: props.idList, AuthToken: Cookies.get('Admin-Token'), webIp: window.location.hostname
  }
  props.typeText === '升级' ? handleUpgrade(params) : handleUninstall(params);;
}
// 升级
const handleUpgrade = (params: any) => {
  agentUpgrade(params).then(res => {
    if (res.data && res.data.code === 200) {
      ElMessage.success(res.data.msg);
      emit('handleCancle', 'success')
    } else {
      ElMessage.error('升级失败，请重试')
      emit('handleCancle')
    }
  })
}
// 卸载
const handleUninstall = (params: any) => {
  agentUninstall(params).then(res => {
    if (res.data && res.data.code === 200) {
      ElMessage.success(res.data.msg);
      emit('handleCancle', 'success')
    } else {
      ElMessage.error('卸载失败，请重试')
      emit('handleCancle')
    }
  })
}
watch(() => props.hintText, (newVal, oldVal) => {
  if (newVal) {
    hintText.value = newVal;
  }
})
</script>

<style scoped lang="scss">
.upContent {
  text-align: left;
  padding-left: 31px;

  &-footer {
    text-align: right;
  }
}
</style>