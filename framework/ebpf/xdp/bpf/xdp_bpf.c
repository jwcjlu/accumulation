#include <linux/bpf.h>
#include <bpf_helpers.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>

#define MAX_BACKENDS 10

struct backend_info {
    u32 ip;
    u16 port;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_BACKENDS);
    __type(key, uint32); // 哈希键（源IP + 源端口）
    __type(value, struct backend_info);
} backend_map SEC(".maps");

SEC("xdp")
int xdp_load_balancer(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    // 解析以太网头
    struct ethhdr *eth = data;
    if (data + sizeof(*eth) > data_end) {
        return XDP_PASS;
    }

    // 只处理IPv4流量
    if (eth->h_proto != htons(ETH_P_IP)) {
        return XDP_PASS;
    }

    // 解析IP头
    struct iphdr *ip = data + sizeof(*eth);
    if (data + sizeof(*eth) + sizeof(*ip) > data_end) {
        return XDP_PASS;
    }

    // 只处理TCP和UDP流量
    if (ip->protocol != IPPROTO_TCP && ip->protocol != IPPROTO_UDP) {
        return XDP_PASS;
    }

    // 计算哈希键（源IP + 源端口）
    __u32 key = ip->saddr;
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + sizeof(*eth) + sizeof(*ip);
        if (data + sizeof(*eth) + sizeof(*ip) + sizeof(*tcp) > data_end) {
            return XDP_PASS;
        }
        key += tcp->source;
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = data + sizeof(*eth) + sizeof(*ip);
        if (data + sizeof(*eth) + sizeof(*ip) + sizeof(*udp) > data_end) {
            return XDP_PASS;
        }
        key += udp->source;
    }

    // 查找后端服务器
    struct backend_info *backend = bpf_map_lookup_elem(&backend_map, &key);
    if (!backend) {
        return XDP_PASS;
    }

    // 修改目的IP和端口
    ip->daddr = backend->ip;
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + sizeof(*eth) + sizeof(*ip);
        tcp->dest = backend->port;
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = data + sizeof(*eth) + sizeof(*ip);
        udp->dest = backend->port;
    }

    // 重新计算IP校验和
    ip->check = 0;
    ip->check = bpf_csum_diff(0, 0, (u16 *)ip, sizeof(*ip), 0);

    return XDP_TX;
}

char __license[] SEC("license") = "GPL";