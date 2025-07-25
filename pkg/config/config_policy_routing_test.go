/*
Copyright 2025 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"net"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestFillLocalRulesFromNode(t *testing.T) {
	testCases := []struct {
		desc                string
		node                *v1.Node
		wantVethGatewayDst  net.IPNet
		wantNodeInternalIPs []net.IP
		wantErr             bool
	}{
		{
			desc: "working case with podCIDR",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: v1.NodeSpec{
					PodCIDR: "10.124.0.0/16",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeInternalIP,
							Address: "10.128.0.24",
						},
					},
				},
			},
			wantVethGatewayDst: net.IPNet{
				IP:   net.IPv4(10, 124, 0, 1),
				Mask: net.CIDRMask(32, 32),
			},
			wantNodeInternalIPs: []net.IP{
				net.IPv4(10, 128, 0, 24),
			},
			wantErr: false,
		},
		{
			desc: "working case with podCIDRs",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: v1.NodeSpec{
					PodCIDRs: []string{"10.124.0.0/16"},
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeInternalIP,
							Address: "10.128.0.24",
						},
					},
				},
			},
			wantVethGatewayDst: net.IPNet{
				IP:   net.IPv4(10, 124, 0, 1),
				Mask: net.CIDRMask(32, 32),
			},
			wantNodeInternalIPs: []net.IP{
				net.IPv4(10, 128, 0, 24),
			},
			wantErr: false,
		},
		{
			desc: "multiple InternalIPs",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: v1.NodeSpec{
					PodCIDR: "10.124.0.0/16",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeInternalIP,
							Address: "10.128.0.24",
						},
						{
							Type:    v1.NodeInternalIP,
							Address: "172.30.0.5",
						},
					},
				},
			},
			wantVethGatewayDst: net.IPNet{
				IP:   net.IPv4(10, 124, 0, 1),
				Mask: net.CIDRMask(32, 32),
			},
			wantNodeInternalIPs: []net.IP{
				net.IPv4(10, 128, 0, 24),
				net.IPv4(172, 30, 0, 5),
			},
			wantErr: false,
		},
		{
			desc: "missing PodCIDR",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeInternalIP,
							Address: "10.128.0.24",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			desc: "missing InternalIP",
			node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
				Spec: v1.NodeSpec{
					PodCIDR: "10.124.0.0/16",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{
						{
							Type:    v1.NodeExternalIP,
							Address: "10.128.0.24",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	originLocalTableRuleConfigs := LocalTableRuleConfigs
	for _, tc := range testCases {
		fakeClient := fake.NewSimpleClientset(tc.node)
		if err := fillLocalRulesFromNode(fakeClient, tc.node.Name); err != nil {
			if !tc.wantErr {
				t.Errorf("fillLocalRulesFromNode() error = %v", err)
			}
			continue
		}
		if !vethGatewayDst.IP.Equal(tc.wantVethGatewayDst.IP) {
			t.Errorf("fillLocalRulesFromNode() vethGatewayDst = %v, want %v", vethGatewayDst, tc.wantVethGatewayDst)
		}
		matchedNodeIPs := len(tc.wantNodeInternalIPs)
		for _, nodeInternalIP := range tc.wantNodeInternalIPs {
			for _, localRule := range LocalTableRuleConfigs {
				if nodeInternalIP.Equal(localRule.(IPRuleConfig).Rule.Dst.IP) {
					matchedNodeIPs--
				}
			}
		}
		if matchedNodeIPs != 0 {
			t.Errorf("fillLocalRulesFromNode() matchedNodeIPDsts = %v, want %v. LocalTableRuleConfigs=%+v", matchedNodeIPs,
				len(tc.wantNodeInternalIPs), LocalTableRuleConfigs)
		}
		// Resetting local configs for testing purpose.
		LocalTableRuleConfigs = originLocalTableRuleConfigs
	}
}
