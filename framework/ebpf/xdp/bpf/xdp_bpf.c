#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <arpa/inet.h> // 包含 htons 函数的声明
#include "xdp_bpf.h"
#define MAX_BACKENDS 10
#define PROXY_IP 0x300030a
#define IP1 0x2d71c065
#define IP2 0x65c0712d
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_BACKENDS);
    __type(key, __u32); // 哈希键（源IP + 源端口）
    __type(value, struct backend_info);
} backend_map SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_BACKENDS);
    __type(key, struct backend_key); // 哈希键（源IP + 源端口）
    __type(value, struct backend_info);
} backend_back_map SEC(".maps");

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
    if (ip->protocol != IPPROTO_TCP){
        return XDP_PASS;
    }

    // 计算哈希键（源IP + 源端口）
    __u32 key=0 ;
     struct backend_key bkey ;
     bkey.bindaddr=ip->daddr;
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + sizeof(*eth) + sizeof(*ip);
        if (data + sizeof(*eth) + sizeof(*ip) + sizeof(*tcp) > data_end) {
            return XDP_PASS;
        }
        key = htons(tcp->dest);
       bkey.bindport=htons(tcp->source);
    }

     if (ip->daddr==IP1){
           bpf_printk("source found: destIP=%x,sourceIP=%x",ip->daddr,ip->saddr);
      }
      if (ip->daddr==IP2){
        bpf_printk("source found: destIP=%x,sourceIP=%x",ip->daddr,ip->saddr);
       }
      if (ip->saddr==IP1){
       bpf_printk("source found: destIP=%x,sourceIP=%x",ip->daddr,ip->saddr);
      }
      if (ip->saddr==IP2){
        bpf_printk("source found: destIP=%x,sourceIP=%x",ip->daddr,ip->saddr);
     }
    struct backend_info *back_backend = bpf_map_lookup_elem(&backend_back_map, &bkey);
     if (back_backend){
       ip->daddr = back_backend->ip;
        if (ip->protocol == IPPROTO_TCP) {
               struct tcphdr *tcp = data + sizeof(*eth) + sizeof(*ip);
               tcp->dest = htons(back_backend->port);
          }
        ip->check = ipv4_csum(ip);
        return XDP_TX;
     }
    // 查找后端服务器
    struct backend_info *backend = bpf_map_lookup_elem(&backend_map, &key);
    if (!backend) {
        return XDP_PASS;
    }
   bpf_printk("Backend found: IP=");
   bpf_printk("%u.", (backend->ip >> 24) & 0xFF);
   bpf_printk("%u.", (backend->ip >> 16) & 0xFF);
   bpf_printk("%u.", (backend->ip >> 8) & 0xFF);
   bpf_printk("%u, ", backend->ip & 0xFF);
   bpf_printk("Backend found: sourceIP=");
   bpf_printk("%u, ", (ip->saddr >> 24) & 0xFF);
   bpf_printk("%u, ", (ip->saddr >> 16) & 0xFF);
   bpf_printk("%u, ", (ip->saddr >> 8) & 0xFF);
   bpf_printk("%u, ", ip->saddr  & 0xFF);
   bpf_printk("Backend found: destIP=");
   bpf_printk("%u, ", (ip->daddr >> 24) & 0xFF);
   bpf_printk("%u, ", (ip->daddr >> 16) & 0xFF);
   bpf_printk("%u, ", (ip->daddr >> 8) & 0xFF);
   bpf_printk("%u, ", ip->daddr  & 0xFF);
   bpf_printk("Port=%u\n", backend->port);
    // 修改目的IP和端口
    ip->daddr = backend->ip;
    struct backend_info new_backend;
    new_backend.ip=ip->daddr;
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = data + sizeof(*eth) + sizeof(*ip);
        tcp->dest = htons(backend->port);
        new_backend.port=  tcp->dest;
        bkey.bindport=backend->port;
    }

    // 重新计算IP校验和
/*    ip->check = 0;
    ip->check = bpf_csum_diff(0, 0, (__be32 *)ip, sizeof(*ip), 0);*/
   /* recalculate IP checksum */
	ip->check = ipv4_csum(ip);
	bpf_map_update_elem(&backend_back_map, &bkey, &new_backend, BPF_ANY);
    return XDP_TX;
}

char __license[] SEC("license") = "GPL";
