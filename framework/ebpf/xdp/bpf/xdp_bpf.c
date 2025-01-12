#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <arpa/inet.h> // 包含 htons 函数的声明

#define MAX_BACKENDS 10

struct backend_info {
    __u32 ip;
    __u16 port;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_BACKENDS);
    __type(key, __u32); // 哈希键（源IP + 源端口）
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
    __u32 key=0 ;
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + sizeof(*eth) + sizeof(*ip);
        if (data + sizeof(*eth) + sizeof(*ip) + sizeof(*tcp) > data_end) {
            return XDP_PASS;
        }
        key += tcp->dest;
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = data + sizeof(*eth) + sizeof(*ip);
        if (data + sizeof(*eth) + sizeof(*ip) + sizeof(*udp) > data_end) {
            return XDP_PASS;
        }
        key += udp->dest;
    }
     bpf_printk("[kye] %x ->\n", key);
    // 查找后端服务器
    struct backend_info *backend = bpf_map_lookup_elem(&backend_map, &key);
    if (!backend) {
        return XDP_PASS;
    }
    bpf_printk("[backend] %x ->\n", backend);
    // 修改目的IP和端口
    ip->daddr = backend->ip;
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + sizeof(*eth) + sizeof(*ip);
        tcp->dest = htons(backend->port);
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = data + sizeof(*eth) + sizeof(*ip);
        udp->dest = htons(backend->port);
    }

    // 重新计算IP校验和
    ip->check = 0;
    ip->check = bpf_csum_diff(0, 0, (__be32 *)ip, sizeof(*ip), 0);

    return XDP_TX;
}

char __license[] SEC("license") = "GPL";
