import { defineStore } from 'pinia';

export const useMacStore = defineStore('mac', {
  state: () => {
    return {
      macIp: '',
    };
  },
  getters: {
    newIp(state) {
      return state.macIp.length > 0 ? state.macIp.split(':')[0] : '';
    }
  },
  actions: {
    setMacIp(ip: string) {
      this.macIp = ip;
      this.newIp;
    }
  }
});

