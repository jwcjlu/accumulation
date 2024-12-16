#include <common.h>

SEC("uprobe/FetchMessage")
int uprobe_FetchMessage(void *ctx) {
  char msg[] = "Hello, World!";
  bpf_printk("invoke FetchMessage: %s\n", msg);
  return 0;
}

char LICENSE[] SEC("license") = "Dual BSD/GPL";

