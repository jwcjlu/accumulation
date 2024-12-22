//go:build ignore

#define __TARGET_ARCH_x86
#define __TARGET_ARCH_x86_64
#include "common.h"
#include "bpf_tracing.h"

char __license[] SEC("license") = "Dual MIT/GPL";
SEC("uretprobe/FetchMessageRet")
int uprobe_FetchMessageRet(struct pt_regs *ctx) {
    char *msg = (char *)PT_REGS_PARM1(ctx);
    bpf_printk("Message:%s\n",msg);
     return 0;
}
