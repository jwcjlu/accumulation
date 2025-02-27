static __always_inline __u16 csum_fold_helper(__u64 csum)
{
	int i;
#pragma unroll
	for (i = 0; i < 4; i++)
	{
		if (csum >> 16)
			csum = (csum & 0xffff) + (csum >> 16);
	}
	return ~csum;
}

static __always_inline __u16 ipv4_csum(struct iphdr *iph)
{
	iph->check = 0;
	unsigned long long csum =
		bpf_csum_diff(0, 0, (unsigned int *)iph, sizeof(struct iphdr), 0);
	return csum_fold_helper(csum);
}
struct backend_info {
    __u32 ip;
    __u16 port;
};


struct backend_key
{
    __u32 bindaddr;
    __u32 bindport;
};