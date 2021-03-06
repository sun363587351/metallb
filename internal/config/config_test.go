package config

import (
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func ipnet(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

func TestParse(t *testing.T) {
	tests := []struct {
		desc string
		raw  string
		want *Config
	}{
		{
			desc: "empty config",
			raw:  "",
			want: &Config{
				Pools: map[string]*Pool{},
			},
		},

		{
			desc: "invalid yaml",
			raw:  "foo:<>$@$2r24j90",
		},

		{
			desc: "config using all features",
			raw: `
peers:
- my-asn: 42
  peer-asn: 142
  peer-address: 1.2.3.4
  peer-port: 1179
  hold-time: 180s
- my-asn: 100
  peer-asn: 200
  peer-address: 2.3.4.5
communities:
  bar: 64512:1234
address-pools:
- name: pool1
  cidr:
  - 10.20.0.0/16
  - 10.50.0.0/24
  avoid-buggy-ips: true
  advertisements:
  - aggregation-length: 32
    localpref: 100
    communities: ["bar", "1234:2345"]
  - aggregation-length: 24
- name: pool2
  cidr:
  - 30.0.0.0/8
`,
			want: &Config{
				Peers: []*Peer{
					{
						MyASN:    42,
						ASN:      142,
						Addr:     net.ParseIP("1.2.3.4"),
						Port:     1179,
						HoldTime: 180 * time.Second,
					},
					{
						MyASN:    100,
						ASN:      200,
						Addr:     net.ParseIP("2.3.4.5"),
						Port:     179,
						HoldTime: 90 * time.Second,
					},
				},
				Pools: map[string]*Pool{
					"pool1": &Pool{
						CIDR:          []*net.IPNet{ipnet("10.20.0.0/16"), ipnet("10.50.0.0/24")},
						AvoidBuggyIPs: true,
						Advertisements: []*Advertisement{
							{
								AggregationLength: 32,
								LocalPref:         100,
								Communities: map[uint32]bool{
									0xfc0004d2: true,
									0x04D20929: true,
								},
							},
							{
								AggregationLength: 24,
								Communities:       map[uint32]bool{},
							},
						},
					},
					"pool2": &Pool{
						CIDR: []*net.IPNet{ipnet("30.0.0.0/8")},
					},
				},
			},
		},

		{
			desc: "peer-only",
			raw: `
peers:
- my-asn: 42
  peer-asn: 42
  peer-address: 1.2.3.4
`,
			want: &Config{
				Peers: []*Peer{
					{
						MyASN:    42,
						ASN:      42,
						Addr:     net.ParseIP("1.2.3.4"),
						Port:     179,
						HoldTime: 90 * time.Second,
					},
				},
				Pools: map[string]*Pool{},
			},
		},

		{
			desc: "invalid peer-address",
			raw: `
peers:
- my-asn: 42
  peer-asn: 42
  peer-address: 1.2.3.400
`,
		},

		{
			desc: "invalid my-asn",
			raw: `
peers:
- peer-asn: 42
  peer-address: 1.2.3.4
`,
		},

		{
			desc: "invalid peer-asn",
			raw: `
peers:
- my-asn: 42
  peer-address: 1.2.3.4
`,
		},

		{
			desc: "invalid hold time (wrong format)",
			raw: `
peers:
- my-asn: 42
  peer-asn: 42
  peer-address: 1.2.3.4
  hold-time: foo
`,
		},

		{
			desc: "invalid hold time (too short)",
			raw: `
peers:
- my-asn: 42
  peer-asn: 42
  peer-address: 1.2.3.4
  hold-time: 1s
`,
		},

		{
			desc: "no pool name",
			raw: `
address-pools:
-
`,
		},

		{
			desc: "address pool with no addresses",
			raw: `
address-pools:
- name: pool1
`,
			want: &Config{
				Pools: map[string]*Pool{
					"pool1": &Pool{},
				},
			},
		},

		{
			desc: "invalid pool CIDR",
			raw: `
address-pools:
- name: pool1
  cidr:
  - 100.200.300.400/24
`,
		},

		{
			desc: "invalid pool CIDR prefix length",
			raw: `
address-pools:
- name: pool1
  cidr:
  - 1.2.3.0/33
`,
		},

		{
			desc: "simple advertisement",
			raw: `
address-pools:
- name: pool1
  advertisements:
  -
`,
			want: &Config{
				Pools: map[string]*Pool{
					"pool1": &Pool{
						Advertisements: []*Advertisement{
							{
								AggregationLength: 32,
								Communities:       map[uint32]bool{},
							},
						},
					},
				},
			},
		},

		{
			desc: "bad aggregation length (too long)",
			raw: `
address-pools:
- name: pool1
  advertisements:
  - aggregation-length: 33
`,
		},

		{
			desc: "bad aggregation length (incompatible with CIDR)",
			raw: `
address-pools:
- name: pool1
  cidr:
  - 10.20.30.40/24
  - 1.2.3.0/28
  advertisements:
  - aggregation-length: 26
`,
		},

		{
			desc: "bad community literal (wrong format)",
			raw: `
address-pools:
- name: pool1
  advertisements:
  - communities: ["1234"]
`,
		},

		{
			desc: "bad community literal (asn part doesn't fit)",
			raw: `
address-pools:
- name: pool1
  advertisements:
  - communities: ["99999999:1"]
`,
		},

		{
			desc: "bad community literal (community# part doesn't fit)",
			raw: `
address-pools:
- name: pool1
  advertisements:
  - communities: ["1:99999999"]
`,
		},

		{
			desc: "bad community ref (unknown ref)",
			raw: `
address-pools:
- name: pool1
  advertisements:
  - communities: ["flarb"]
`,
		},

		{
			desc: "bad community ref (ref asn doesn't fit)",
			raw: `
communities:
  flarb: 99999999:1
address-pools:
- name: pool1
  advertisements:
  - communities: ["flarb"]
`,
		},

		{
			desc: "bad community ref (ref community# doesn't fit)",
			raw: `
communities:
  flarb: 1:99999999
address-pools:
- name: pool1
  advertisements:
  - communities: ["flarb"]
`,
		},

		{
			desc: "duplicate pool definition",
			raw: `
address-pools:
- name: pool1
- name: pool1
- name: pool2
`,
		},

		{
			desc: "duplicate CIDRs",
			raw: `
address-pools:
- name: pool1
  cidr:
  - 10.0.0.0/8
- name: pool2
  cidr:
  - 10.0.0.0/8
`,
		},

		{
			desc: "overlapping CIDRs",
			raw: `
address-pools:
- name: pool1
  cidr:
  - 10.0.0.0/8
- name: pool2
  cidr:
  - 10.0.0.0/16
`,
		},
	}

	for _, test := range tests {
		got, err := Parse([]byte(test.raw))
		if err != nil && test.want != nil {
			t.Errorf("%q: parse failed: %s", test.desc, err)
			continue
		}
		if test.want == nil && err == nil {
			t.Errorf("%q: parse unexpectedly succeeded", test.desc)
			continue
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("%q: parse returned wrong result (-want, +got)\n%s", test.desc, diff)
		}
	}
}
