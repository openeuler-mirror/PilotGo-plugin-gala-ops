<template>
  <div class="pm_table">
    <!-- table工具条 -->
    <el-row class="pm_header">
      <slot name="tool_bar"></slot>
    </el-row>
    <!-- 列表 -->
    <div class="pm_table_content" ref="tableBox">
      <el-table ref="table" :data="tableData" class="table" @selection-change="handleSelectionChange" v-loading="loading">
        <slot></slot>
        <template #append>
          <slot name="append"></slot>
        </template>
        <template #empty>
          <el-empty description="暂无数据"></el-empty>
        </template>
      </el-table>
    </div>
    <!-- 分页 -->
    <div class="pm_table_page">
      <el-config-provider :locale="zhCn">
        <el-pagination v-model:current-page="page.currentPage" v-model:page-size="page.pageSize"
          :page-sizes="[20, 25, 50, 75, 100]" :small="page.small" :background="page.background"
          layout="total, sizes, prev, pager, next, jumper" :total="page.total" @size-change="getTableData"
          @current-change="getTableData" />
      </el-config-provider>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
const props = defineProps({
  getData: {
    type: Function,
    required: true
  }
})
const emit = defineEmits(["handleSelect"])
const loading = ref(false);
const tableData = ref([] as any);
const currentNum = ref(0); //复选框当前页数量
const checkedIps = ref([] as string[])
const page = reactive({
  total: 0,
  currentPage: 1,
  pageSize: 20,
  small: true,
  background: true,
})

onMounted(async () => {
  await getTableData();
})

// 获取表格数据
const getTableData = () => {
  loading.value = true;
  props.getData!({ page: page.currentPage, size: page.pageSize }).then((res: any) => {
    if (res.data && res.data.code === 200) {
      loading.value = false;
      tableData.value = res.data.data;
      currentNum.value = res.data.data.length;
      page.total = res.data.total;
    } else {
      loading.value = false;
      tableData.value = [];
      currentNum.value = 0;
      page.total = 10;
    }
  });
}


// 表格复选框
const handleSelectionChange = (val: any[]) => {
  // 处理选中的行
  checkedIps.value = [];
  const checkedIds = [] as any;
  if (val) {
    val.forEach(item => {
      checkedIps.value.push(item.ip);
      checkedIds.push(item.id);
    });
    emit('handleSelect', checkedIds)
  }

}
</script>

<style lang="scss" scoped>
.pm_table {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;


  &_content {
    height: 80%;

    .table {
      width: 100%;
      height: 100%;
    }
  }

  &_page {
    height: 20px;
    display: flex;
    justify-content: flex-end;
  }


}
</style>
