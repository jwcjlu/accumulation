//go:build ignore

#include "common.h"

char __license[] SEC("license") = "Dual MIT/GPL";

struct event {
	u32 pid;
	u8 line[80];
};

SEC("uretprobe/FetchMessage")
int uprobe_FetchMessage(struct pt_regs *ctx) {
        char msg[] = "your_message_here";

    // Increment the counter
    static int index = 0;
    index++;

    // Print the message
    char buf[256];
    bpf_probe_read_user_str(buf, sizeof(buf), msg);
    bpf_printk("hello world %d! Message: %s\n", index, buf);
     return 0;
}


