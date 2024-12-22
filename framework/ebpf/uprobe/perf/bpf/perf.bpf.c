#define __TARGET_ARCH_x86
#define __TARGET_ARCH_x86_64
#include "common.h"
#include "bpf_tracing.h"
#include <uapi/linux/perf_event.h>
#include <linux/sched.h>
#define MAX_CONCURRENT 50
struct func_perf_stat_t {
    u64 start_time;
    u64 end_time;
    u32 pid;
    u8 msg[80];
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, u32); // 修改 key 类型为 u64
    __type(value, struct func_perf_stat_t);
    __uint(max_entries, MAX_CONCURRENT);
} func_perf_stat_events SEC(".maps");

char __license[] SEC("license") = "Dual MIT/GPL";
struct {
	__uint(type, BPF_MAP_TYPE_PERF_EVENT_ARRAY);
} events SEC(".maps");

SEC("uprobe/FetchMessage")
int uprobe_FetchMessage(struct pt_regs *ctx) {
    struct func_perf_stat_t func_perf_stat = {};
    func_perf_stat.start_time = bpf_ktime_get_ns();
    func_perf_stat.pid = bpf_get_current_pid_tgid() >> 32; // 获取当前进程 PID
    bpf_probe_read(&func_perf_stat.msg, sizeof(func_perf_stat.msg), (void *)PT_REGS_RC(ctx));
    bpf_map_update_elem(&func_perf_stat_events, &func_perf_stat.pid, &func_perf_stat, 0);
    return 0;
}

SEC("uretprobe/FetchMessage")
int uretprobe_FetchMessage(struct pt_regs *ctx) {
    u32 key = bpf_get_current_pid_tgid() >> 32;
    struct func_perf_stat_t *func_perf_stat = bpf_map_lookup_elem(&func_perf_stat_events, &key);
    if (func_perf_stat == NULL) {
        bpf_printk("perf_request is null\n");
        return -1; // 返回错误码
    }
    func_perf_stat->end_time = bpf_ktime_get_ns();
    bpf_map_delete_elem(&func_perf_stat_events, &key);
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, func_perf_stat, sizeof(*func_perf_stat));
    return 0;
}

