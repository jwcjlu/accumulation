//go:build ignore

#include "uprobe.h"


char __license[] SEC("license") = "Dual MIT/GPL";

SEC("uretprobe/FetchMessage")
int uprobe_FetchMessage(struct pt_regs *ctx) {
      // 打印日志或执行其他操作
   void *writer = get_argument(ctx, 1);
   bpf_printk("invoke uprobe_FetchMessage: %s\n", writer);
     return 0;
}

