package controllers

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type AWSTestCase struct {
	// Original document policy.
	policy string
	// Expected document policy
	expectedPolicy string
	// service account namespace
	serviceAccountNamespace string
	// service account name
	serviceAccountName string
}

func TestAddIAMRoleAnnotation(t *testing.T) {
	roleName := "arn:aws:iam::34892524:role/s3-reader"
	sa := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
			Name:      "sa",
		},
	}

	addIAMRoleAnnotation(sa, roleName)
	assert.NotNil(t, sa.Annotations)
	assert.Equal(t, roleName, sa.Annotations[AWS_ANNOTATION_KEY])
}

func TestRemoveIAMRoleAnnotation(t *testing.T) {
	roleName := "arn:aws:iam::34892524:role/s3-reader"
	sa := &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Namespace:   "default",
			Name:        "sa",
			Annotations: map[string]string{AWS_ANNOTATION_KEY: roleName},
		},
	}

	addIAMRoleAnnotation(sa, roleName)
	assert.NotNil(t, sa.Annotations)
	assert.Empty(t, sa.Annotations[AWS_DEFAULT_AUDIENCE])
}

func TestGetIssuerUrlFromRoleArn(t *testing.T) {
	issuerUrl := "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
	arn := fmt.Sprintf("arn:aws:iam::34892524:oidc-provider/%v", issuerUrl)
	result := getIssuerUrlFromProviderArn(arn)
	assert.Equal(t, result, issuerUrl)
}

func TestGetIAMRoleNameFromIAMRoleArn(t *testing.T) {
	roleName := "test-iam-role"
	arn := fmt.Sprintf("arn:aws:iam::34892524:role/%v", roleName)
	result := getIAMRoleNameFromIAMRoleArn(arn)
	assert.Equal(t, result, roleName)
}

func TestAddServiceAccountInAssumeRolePolicy(t *testing.T) {
	tests := []AWSTestCase{
		{
			policy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": "sts:AssumeRoleWithWebIdentity",
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"]
						}
					  }
					}
				  ]
				}
				`,
			expectedPolicy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": "sts:AssumeRoleWithWebIdentity",
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub": ["system:serviceaccount:ns1:sa1"]
						}
					  }
					}
				  ]
				}
				`,
			serviceAccountNamespace: "ns1",
			serviceAccountName:      "sa1",
		},
		{
			policy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": "sts:AssumeRoleWithWebIdentity",
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
							"oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub":[]
						}
					  }
					}
				  ]
				}
				`,
			expectedPolicy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": "sts:AssumeRoleWithWebIdentity",
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub": ["system:serviceaccount:ns1:sa1"]
						}
					  }
					}
				  ]
				}
				`,
			serviceAccountNamespace: "ns1",
			serviceAccountName:      "sa1",
		},
		{
			policy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": "sts:AssumeRoleWithWebIdentity",
					  "Condition": {
						"StringEquals": {
						  	"oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
							"oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub":["system:serviceaccount:ns1:sa2"]
						}
					  }
					}
				  ]
				}
				`,
			expectedPolicy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": "sts:AssumeRoleWithWebIdentity",
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub": ["system:serviceaccount:ns1:sa2", "system:serviceaccount:ns1:sa1"]
						}
					  }
					}
				  ]
				}
				`,
			serviceAccountNamespace: "ns1",
			serviceAccountName:      "sa1",
		},
	}

	for _, test := range tests {
		result, _ := addServiceAccountInAssumeRolePolicy(test.policy, test.serviceAccountNamespace, test.serviceAccountName)
		require.JSONEq(t, test.expectedPolicy, result)
	}
}

func TestRemoveServiceAccountInAssumeRolePolicy(t *testing.T) {
	tests := []AWSTestCase{
		{
			policy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": [
						"sts:AssumeRoleWithWebIdentity"
					  ],
					  "Condition": {
						"StringEquals": {
 						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub": ["system:serviceaccount:ns1:sa1", "system:serviceaccount:ns1:sa2"]
						}
					  }
					}
				  ]
				}`,
			expectedPolicy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": [
						"sts:AssumeRoleWithWebIdentity"
					  ],
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub": ["system:serviceaccount:ns1:sa1"]
						}
					  }
					}
				  ]
				}`,
			serviceAccountNamespace: "ns1",
			serviceAccountName:      "sa2",
		},
		{
			policy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": [
						"sts:AssumeRoleWithWebIdentity"
					  ],
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"],
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:sub": ["system:serviceaccount:ns1:sa2"]
						}
					  }
					}
				  ]
				}`,
			expectedPolicy: `
				{
				  "Version": "2012-10-17",
				  "Statement": [
					{
					  "Effect": "Allow",
					  "Principal": {
						"Federated": "arn:aws:iam::34892524:oidc-provider/oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72"
					  },
					  "Action": [
						"sts:AssumeRoleWithWebIdentity"
					  ],
					  "Condition": {
						"StringEquals": {
						  "oidc.beta.us-west-2.wesley.amazonaws.com/id/50D94CFC65139194EDC21891B611EF72:aud": ["sts.amazonaws.com"]
						}
					  }
					}
				  ]
				}`,
			serviceAccountNamespace: "ns1",
			serviceAccountName:      "sa2",
		},
	}

	for _, test := range tests {
		result, _ := removeServiceAccountInAssumeRolePolicy(test.policy, test.serviceAccountNamespace, test.serviceAccountName)
		require.JSONEq(t, test.expectedPolicy, result)
	}
}

func TestIsAnnotateOnly(t *testing.T) {
	aws := &AwsIAMForServiceAccount{AnnotateOnly: true}

	// Check that the result is true
	assert.True(t, aws.isAnnotateOnly())

	aws = &AwsIAMForServiceAccount{AnnotateOnly: false}
	// Check that the result is true
	assert.False(t, aws.isAnnotateOnly())
}

// ==============================================================================
// GitHub Issue #453 Regression Test Matrix (12 Test Cases)
// ==============================================================================

// Case 1: Single Kubeflow OIDC statement (baseline)
func TestIssue453_Case1_SingleKubeflowOIDCStatement(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:kubeflow-user:default-editor"]
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "kubeflow-user", "default-editor")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Case 2: Kubeflow statement at index 1 (ordering independence)
func TestIssue453_Case2_KubeflowStatementAtIndex1(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "UnrelatedStatement",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:role/other-role"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "KubeflowStatement",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": ["sts.amazonaws.com"]
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "UnrelatedStatement",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:role/other-role"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "KubeflowStatement",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": ["sts.amazonaws.com"],
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:ns1:sa1"]
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "ns1", "sa1")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Case 3: Unrelated AWS trust statement preserved
func TestIssue453_Case3_UnrelatedAWSTrustStatement(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "AWSTrust",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:root"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "EKSIRSA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:ns1:sa1"]
					}
				}
			}
		]
	}`

	// Remove SA: AWS trust statement must remain
	removeResult, err := removeServiceAccountInAssumeRolePolicy(policy, "ns1", "sa1")
	require.NoError(t, err)

	expectedRemove := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "AWSTrust",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:root"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "EKSIRSA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`
	require.JSONEq(t, expectedRemove, removeResult)
}

// Case 4: Unrelated OIDC statement before Kubeflow (GitHub Actions)
func TestIssue453_Case4_UnrelatedOIDCBeforeKubeflow(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "GitHubActionsOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
						"token.actions.githubusercontent.com:sub": "repo:my-org/my-repo:ref:refs/heads/main"
					}
				}
			},
			{
				"Sid": "KubeflowOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "GitHubActionsOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
						"token.actions.githubusercontent.com:sub": "repo:my-org/my-repo:ref:refs/heads/main"
					}
				}
			},
			{
				"Sid": "KubeflowOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:kubeflow-test:default-editor"]
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "kubeflow-test", "default-editor")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Case 5: Unrelated OIDC statement after Kubeflow (GitHub Actions)
func TestIssue453_Case5_UnrelatedOIDCAfterKubeflow(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "KubeflowOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					}
				}
			},
			{
				"Sid": "GitHubActionsOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
						"token.actions.githubusercontent.com:sub": "repo:my-org/my-repo:ref:refs/heads/main"
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "KubeflowOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:kubeflow-test:default-editor"]
					}
				}
			},
			{
				"Sid": "GitHubActionsOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
						"token.actions.githubusercontent.com:sub": "repo:my-org/my-repo:ref:refs/heads/main"
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "kubeflow-test", "default-editor")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Case 6: Multiple OIDC statements disambiguated by existing namespace service accounts
func TestIssue453_Case6_MultipleOIDCStatementsDisambiguated(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "EKSClusterA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A:sub": ["system:serviceaccount:ns-alpha:editor"]
					}
				}
			},
			{
				"Sid": "EKSClusterB",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B:sub": ["system:serviceaccount:ns-beta:viewer"]
					}
				}
			}
		]
	}`

	// Adding sa to ns-alpha: must choose ClusterA
	addResult, err := addServiceAccountInAssumeRolePolicy(policy, "ns-alpha", "viewer")
	require.NoError(t, err)

	expectedAdd := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "EKSClusterA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A:sub": [
							"system:serviceaccount:ns-alpha:editor",
							"system:serviceaccount:ns-alpha:viewer"
						]
					}
				}
			},
			{
				"Sid": "EKSClusterB",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B:sub": ["system:serviceaccount:ns-beta:viewer"]
					}
				}
			}
		]
	}`
	require.JSONEq(t, expectedAdd, addResult)
}

// Case 7: Ambiguous multiple OIDC statements -> safe failure with policy unchanged
func TestIssue453_Case7_AmbiguousMultipleOIDCStatements(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "ClusterA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A:aud": "sts.amazonaws.com"
					}
				}
			},
			{
				"Sid": "ClusterB",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "new-ns", "default-editor")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous trust policy")
	// Policy must be returned unchanged
	require.JSONEq(t, policy, result)
}

// Case 8: Existing custom fields on target statement are preserved
func TestIssue453_Case8_ExistingCustomFieldsOnTargetStatementPreserved(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "CustomMySid123",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": [
					"sts:AssumeRoleWithWebIdentity"
				],
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					},
					"StringLike": {
						"aws:PrincipalTag/Department": "MachineLearning"
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "CustomMySid123",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": [
					"sts:AssumeRoleWithWebIdentity"
				],
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:ns1:sa1"]
					},
					"StringLike": {
						"aws:PrincipalTag/Department": "MachineLearning"
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "ns1", "sa1")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Case 9: Add preserves every unrelated statement
func TestIssue453_Case9_AddPreservesEveryUnrelatedStatement(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "IAMRoleTrust",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:role/admin-role"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "SAMLTrust",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:saml-provider/MyOkta"
				},
				"Action": "sts:AssumeRoleWithSAML",
				"Condition": {
					"StringEquals": {
						"SAML:aud": "https://signin.aws.amazon.com/saml"
					}
				}
			},
			{
				"Sid": "KubeflowIRSA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "IAMRoleTrust",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:role/admin-role"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "SAMLTrust",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:saml-provider/MyOkta"
				},
				"Action": "sts:AssumeRoleWithSAML",
				"Condition": {
					"StringEquals": {
						"SAML:aud": "https://signin.aws.amazon.com/saml"
					}
				}
			},
			{
				"Sid": "KubeflowIRSA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:ns1:sa1"]
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "ns1", "sa1")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Case 10: Remove preserves every unrelated statement
func TestIssue453_Case10_RemovePreservesEveryUnrelatedStatement(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "IAMRoleTrust",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:role/admin-role"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "SAMLTrust",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:saml-provider/MyOkta"
				},
				"Action": "sts:AssumeRoleWithSAML",
				"Condition": {
					"StringEquals": {
						"SAML:aud": "https://signin.aws.amazon.com/saml"
					}
				}
			},
			{
				"Sid": "KubeflowIRSA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:ns1:sa1"]
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "IAMRoleTrust",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:role/admin-role"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "SAMLTrust",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:saml-provider/MyOkta"
				},
				"Action": "sts:AssumeRoleWithSAML",
				"Condition": {
					"StringEquals": {
						"SAML:aud": "https://signin.aws.amazon.com/saml"
					}
				}
			},
			{
				"Sid": "KubeflowIRSA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`

	result, err := removeServiceAccountInAssumeRolePolicy(policy, "ns1", "sa1")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Case 11: Add is idempotent
func TestIssue453_Case11_AddIdempotency(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:ns1:sa1"]
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "ns1", "sa1")
	require.Error(t, err)
	assert.IsType(t, &ConditionExistError{}, err)
	require.JSONEq(t, policy, result)
}

// Case 12: Remove absent service account leaves policy intact
func TestIssue453_Case12_RemoveAbsentServiceAccount(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:ns1:sa1"]
					}
				}
			}
		]
	}`

	result, err := removeServiceAccountInAssumeRolePolicy(policy, "ns1", "absent-sa")
	require.NoError(t, err)
	require.JSONEq(t, policy, result)
}

// ==============================================================================
// Additional Adversarial Multi-OIDC & Symmetry Tests
// ==============================================================================

// Adversarial Test: GitLab CI OIDC provider alongside Kubeflow EKS OIDC
func TestIssue453_Adversarial_GitLabCIOIDC(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "GitLabCIOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/gitlab.com"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"gitlab.com:aud": "sts.amazonaws.com",
						"gitlab.com:sub": "project_path:my-group/my-project:ref_type:branch:ref:main"
					}
				}
			},
			{
				"Sid": "KubeflowEKS",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`

	expected := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "GitLabCIOIDC",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/gitlab.com"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"gitlab.com:aud": "sts.amazonaws.com",
						"gitlab.com:sub": "project_path:my-group/my-project:ref_type:branch:ref:main"
					}
				}
			},
			{
				"Sid": "KubeflowEKS",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/11223344"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/11223344:sub": ["system:serviceaccount:kf-user:default-editor"]
					}
				}
			}
		]
	}`

	result, err := addServiceAccountInAssumeRolePolicy(policy, "kf-user", "default-editor")
	require.NoError(t, err)
	require.JSONEq(t, expected, result)
}

// Adversarial Test: Multiple EKS statements with overlapping namespace subjects -> safe failure
func TestIssue453_Adversarial_MultipleEKSOverlappingNamespace(t *testing.T) {
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "ClusterA",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_A:sub": ["system:serviceaccount:shared-ns:sa-one"]
					}
				}
			},
			{
				"Sid": "ClusterB",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B:aud": "sts.amazonaws.com",
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_B:sub": ["system:serviceaccount:shared-ns:sa-two"]
					}
				}
			}
		]
	}`

	// Adding sa-three to shared-ns: both clusters already have service accounts for shared-ns
	result, err := addServiceAccountInAssumeRolePolicy(policy, "shared-ns", "sa-three")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous trust policy: multiple candidate OIDC statements contain service accounts for namespace shared-ns")
	// Policy must be returned untouched
	require.JSONEq(t, policy, result)
}

// Adversarial Test: Add and Remove roundtrip symmetry
func TestIssue453_Adversarial_AddRemoveRoundTrip(t *testing.T) {
	originalPolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "IAMTrust",
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:iam::123456789012:role/ci-role"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "GHActions",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
						"token.actions.githubusercontent.com:sub": "repo:org/repo:ref:refs/heads/main"
					}
				}
			},
			{
				"Sid": "KubeflowEKS",
				"Effect": "Allow",
				"Principal": {
					"Federated": "arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_1"
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"oidc.eks.us-west-2.amazonaws.com/id/CLUSTER_1:aud": "sts.amazonaws.com"
					}
				}
			}
		]
	}`

	// Step 1: Add service account
	addedPolicy, err := addServiceAccountInAssumeRolePolicy(originalPolicy, "my-ns", "editor")
	require.NoError(t, err)

	// Step 2: Remove same service account
	removedPolicy, err := removeServiceAccountInAssumeRolePolicy(addedPolicy, "my-ns", "editor")
	require.NoError(t, err)

	// Result must be strictly identical to originalPolicy
	require.JSONEq(t, originalPolicy, removedPolicy)
}
