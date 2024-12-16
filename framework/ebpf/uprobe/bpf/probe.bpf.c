//go:build ignore

#include "common.h"


char __license[] SEC("license") = "Dual MIT/GPL";

SEC("uretprobe/FetchMessage")
int uprobe_FetchMessage(void *ctx) {
      // 打印日志或执行其他操作
   char msg[] = "Hello, World!";
   bpf_printk("invoke uprobe_FetchMessage: %s\n", msg);
     return 0;
}

