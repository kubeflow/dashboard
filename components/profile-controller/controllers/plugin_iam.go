package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/go-logr/logr"
	profilev1 "github.com/kubeflow/dashboard/components/profile-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// plugin kind
	KIND_AWS_IAM_FOR_SERVICE_ACCOUNT = "AwsIamForServiceAccount"
	AWS_ANNOTATION_KEY               = "eks.amazonaws.com/role-arn"
	AWS_TRUST_IDENTITY_SUBJECT       = "system:serviceaccount:%s:%s"
	AWS_DEFAULT_AUDIENCE             = "sts.amazonaws.com"
	DEFAULT_SERVICE_ACCOUNT          = DEFAULT_EDITOR
)

type AwsIAMForServiceAccount struct {
	AwsIAMRole   string `json:"awsIamRole,omitempty"`
	AnnotateOnly bool   `json:"annotateOnly,omitempty"`
}

// ApplyPlugin annotate service account with the ARN of the IAM role and update trust relationship of IAM role
func (aws *AwsIAMForServiceAccount) ApplyPlugin(r *ProfileReconciler, profile *profilev1.Profile) error {
	logger := r.Log.WithValues("profile", profile.Name)
	if err := aws.patchAnnotation(r, profile.Name, DEFAULT_SERVICE_ACCOUNT, addIAMRoleAnnotation, logger); err != nil {
		return err
	}

	logger.Info("Setting up iam roles and policy for service account.", "ServiceAccount", DEFAULT_SERVICE_ACCOUNT, "Role", aws.AwsIAMRole)
	return aws.updateIAMForServiceAccount(profile.Name, DEFAULT_SERVICE_ACCOUNT, addServiceAccountInAssumeRolePolicy, logger)
}

// RevokePlugin remove role in service account annotation and delete service account record in IAM trust relationship.
func (aws *AwsIAMForServiceAccount) RevokePlugin(r *ProfileReconciler, profile *profilev1.Profile) error {
	logger := r.Log.WithValues("profile", profile.Name)
	if err := aws.patchAnnotation(r, profile.Name, DEFAULT_SERVICE_ACCOUNT, removeIAMRoleAnnotation, logger); err != nil {
		return err
	}

	logger.Info("Clean up AWS IAM Role for Service Account.", "ServiceAccount", DEFAULT_SERVICE_ACCOUNT, "Role", aws.AwsIAMRole)
	return aws.updateIAMForServiceAccount(profile.Name, DEFAULT_SERVICE_ACCOUNT, removeServiceAccountInAssumeRolePolicy, logger)
}

// patchAnnotation will patch annotation to k8s service account in order to pair up with GCP identity
func (aws *AwsIAMForServiceAccount) patchAnnotation(r *ProfileReconciler, namespace string, ksa string, annotationFunc func(*corev1.ServiceAccount, string), logger logr.Logger) error {
	ctx := context.Background()
	found := &corev1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: ksa, Namespace: namespace}, found)
	if err != nil {
		return err
	}

	if aws.AwsIAMRole == "" {
		return errors.New("failed to setup service account because awsIamRole is empty")
	}

	annotationFunc(found, aws.AwsIAMRole)
	logger.Info("Patch Annotation for service account: ", "namespace ", namespace, "name ", ksa)
	return r.Update(ctx, found)
}

// updateIAMForServiceAccount update AWS IAM Roles trust relationship with namespace and service account
func (aws *AwsIAMForServiceAccount) updateIAMForServiceAccount(serviceAccountNamespace, serviceAccountName string, updateAssumeRolePolicy func(string, string, string) (string, error), logger logr.Logger) error {
	if aws.isAnnotateOnly() {
		logger.Info("AnnotateOnly set to true IAM roles and policy will not be mutated")
		return nil
	}

	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("error loading AWS config: %v", err)
	}
	svc := iam.NewFromConfig(cfg)
	roleName := getIAMRoleNameFromIAMRoleArn(aws.AwsIAMRole)
	roleInput := &iam.GetRoleInput{
		RoleName: awssdk.String(roleName),
	}

	output, err := svc.GetRole(ctx, roleInput)
	if err != nil {
		return err
	}

	// Seems AssumeRolePolicyDocument is URL encoded
	decodeValue, err := url.QueryUnescape(awssdk.ToString(output.Role.AssumeRolePolicyDocument))
	if err != nil {
		return err
	}

	updatedRolePolicy, err := updateAssumeRolePolicy(decodeValue, serviceAccountNamespace, serviceAccountName)
	if err != nil {
		if _, ok := err.(*ConditionExistError); ok {
			// we just skip role update here
			return nil
		}
	}

	input := &iam.UpdateAssumeRolePolicyInput{
		RoleName:       awssdk.String(roleName),
		PolicyDocument: awssdk.String(updatedRolePolicy),
	}
	if _, err = svc.UpdateAssumeRolePolicy(ctx, input); err != nil {
		return err
	}
	return nil
}

// addIAMRoleAnnotation add `eks.amazonaws.com/role-arn:roleArn` to service account annotations
func addIAMRoleAnnotation(sa *corev1.ServiceAccount, iamRoleArn string) {
	if sa.Annotations == nil {
		sa.Annotations = map[string]string{AWS_ANNOTATION_KEY: iamRoleArn}
	} else {
		sa.Annotations[AWS_ANNOTATION_KEY] = iamRoleArn
	}
}

// removeIAMRoleAnnotation remove `eks.amazonaws.com/role-arn:roleArn` in service account annotations
func removeIAMRoleAnnotation(sa *corev1.ServiceAccount, iamRoleArn string) {
	if sa.Annotations == nil {
	} else {
		if _, ok := sa.Annotations[AWS_ANNOTATION_KEY]; ok {
			delete(sa.Annotations, AWS_ANNOTATION_KEY)
		}
	}
}

func statementHasAction(statement MapOfInterfaces, targetAction string) bool {
	action, ok := statement["Action"]
	if !ok {
		return false
	}
	switch a := action.(type) {
	case string:
		return strings.EqualFold(a, targetAction)
	case []interface{}:
		for _, item := range a {
			if str, ok := item.(string); ok && strings.EqualFold(str, targetAction) {
				return true
			}
		}
	case []string:
		for _, item := range a {
			if strings.EqualFold(item, targetAction) {
				return true
			}
		}
	}
	return false
}

func getStatementFederatedArn(statement MapOfInterfaces) string {
	principal, ok := statement["Principal"]
	if !ok {
		return ""
	}
	pMap, ok := principal.(map[string]interface{})
	if !ok {
		return ""
	}
	fed, ok := pMap["Federated"]
	if !ok {
		return ""
	}
	fedStr, ok := fed.(string)
	if !ok {
		return ""
	}
	return fedStr
}

func stringEqualsMatches(val interface{}, target string) bool {
	switch v := val.(type) {
	case string:
		return v == target
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok && str == target {
				return true
			}
		}
	case []string:
		for _, item := range v {
			if item == target {
				return true
			}
		}
	}
	return false
}

func extractSubValues(val interface{}) []string {
	var res []string
	if val == nil {
		return res
	}
	switch v := val.(type) {
	case string:
		if v != "" {
			res = append(res, v)
		}
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				res = append(res, str)
			}
		}
	case []string:
		for _, str := range v {
			if str != "" {
				res = append(res, str)
			}
		}
	}
	return res
}

type oidcCandidate struct {
	index               int
	providerArn         string
	issuerUrl           string
	existingSubs        []string
	containsTargetSA    bool
	containsNamespaceSA bool
}

func findKubeflowOIDCStatementIndex(statements []MapOfInterfaces, namespace, serviceAccountName string, isRemove bool) (int, string, error) {
	targetSA := fmt.Sprintf(AWS_TRUST_IDENTITY_SUBJECT, namespace, serviceAccountName)
	nsPrefix := fmt.Sprintf("system:serviceaccount:%s:", namespace)

	var candidates []oidcCandidate

	for i, stmt := range statements {
		fedArn := getStatementFederatedArn(stmt)
		if fedArn == "" || !strings.Contains(fedArn, "oidc-provider/") {
			continue
		}
		if !statementHasAction(stmt, "sts:AssumeRoleWithWebIdentity") {
			continue
		}
		if eff, ok := stmt["Effect"].(string); ok && eff != "Allow" {
			continue
		}

		issuerUrl := getIssuerUrlFromProviderArn(fedArn)
		audKey := issuerUrl + ":aud"
		subKey := issuerUrl + ":sub"

		condMap, ok := stmt["Condition"].(map[string]interface{})
		if !ok {
			continue
		}
		strEqualsMap, ok := condMap["StringEquals"].(map[string]interface{})
		if !ok {
			continue
		}

		// Must have aud == sts.amazonaws.com
		if !stringEqualsMatches(strEqualsMap[audKey], AWS_DEFAULT_AUDIENCE) {
			continue
		}

		// Extract existing subjects
		subs := extractSubValues(strEqualsMap[subKey])

		// Check if it has non-Kubernetes subjects (e.g. repo:org/repo:... for GitHub Actions)
		hasNonK8sSubject := false
		for _, sub := range subs {
			if !strings.HasPrefix(sub, "system:serviceaccount:") {
				hasNonK8sSubject = true
				break
			}
		}
		if hasNonK8sSubject {
			// Disqualify third-party non-Kubernetes OIDC statements
			continue
		}

		cand := oidcCandidate{
			index:        i,
			providerArn:  fedArn,
			issuerUrl:    issuerUrl,
			existingSubs: subs,
		}

		for _, sub := range subs {
			if sub == targetSA {
				cand.containsTargetSA = true
			}
			if strings.HasPrefix(sub, nsPrefix) {
				cand.containsNamespaceSA = true
			}
		}

		candidates = append(candidates, cand)
	}

	if isRemove {
		var matched []oidcCandidate
		for _, c := range candidates {
			if c.containsTargetSA {
				matched = append(matched, c)
			}
		}
		if len(matched) == 0 {
			// Target SA is not present in any candidate statement; this is an idempotent remove
			return -1, "", nil
		}
		if len(matched) == 1 {
			return matched[0].index, matched[0].providerArn, nil
		}
		return -1, "", fmt.Errorf("ambiguous trust policy: multiple candidate OIDC statements contain service account %s", targetSA)
	}

	// For Add:
	// 1. Check idempotency: if target SA already present in any candidate statement
	for _, c := range candidates {
		if c.containsTargetSA {
			return -1, "", &ConditionExistError{msg: fmt.Sprintf("service account %s already exists in trust policy", targetSA)}
		}
	}

	// 2. If exactly one candidate already manages service accounts for this namespace
	var nsMatched []oidcCandidate
	for _, c := range candidates {
		if c.containsNamespaceSA {
			nsMatched = append(nsMatched, c)
		}
	}
	if len(nsMatched) == 1 {
		return nsMatched[0].index, nsMatched[0].providerArn, nil
	}
	if len(nsMatched) > 1 {
		return -1, "", fmt.Errorf("ambiguous trust policy: multiple candidate OIDC statements contain service accounts for namespace %s", namespace)
	}

	// 3. Ambiguity evaluation on candidates
	if len(candidates) == 0 {
		return -1, "", errors.New("no matching OIDC trust statement found in trust policy")
	}
	if len(candidates) == 1 {
		return candidates[0].index, candidates[0].providerArn, nil
	}

	// Multiple candidates exist and neither can be uniquely proven to be the Kubeflow statement
	return -1, "", fmt.Errorf("ambiguous trust policy: found %d candidate OIDC statements; cannot safely determine Kubeflow-managed statement", len(candidates))
}

// add serviceAccountNamespace/serviceAccountName in assumeRolePolicy
func addServiceAccountInAssumeRolePolicy(policyDocument, serviceAccountNamespace, serviceAccountName string) (string, error) {
	var oldDoc MapOfInterfaces
	err := json.Unmarshal([]byte(policyDocument), &oldDoc)
	if err != nil {
		return "", err
	}
	var statements []MapOfInterfaces
	statementInBytes, err := json.Marshal(oldDoc["Statement"])
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(statementInBytes, &statements); err != nil {
		return "", err
	}

	targetIdx, oidcRoleArn, err := findKubeflowOIDCStatementIndex(statements, serviceAccountNamespace, serviceAccountName, false)
	if err != nil {
		return policyDocument, err
	}

	issuerUrlWithProtocol := getIssuerUrlFromProviderArn(oidcRoleArn)
	subKey := fmt.Sprintf("%s:sub", issuerUrlWithProtocol)
	audKey := fmt.Sprintf("%s:aud", issuerUrlWithProtocol)
	trustIdentity := fmt.Sprintf(AWS_TRUST_IDENTITY_SUBJECT, serviceAccountNamespace, serviceAccountName)

	stmt := statements[targetIdx]

	condObj, ok := stmt["Condition"]
	var condMap map[string]interface{}
	if ok && condObj != nil {
		condMap, _ = condObj.(map[string]interface{})
	}
	if condMap == nil {
		condMap = make(map[string]interface{})
		stmt["Condition"] = condMap
	}

	strEqObj, ok := condMap["StringEquals"]
	var strEqMap map[string]interface{}
	if ok && strEqObj != nil {
		strEqMap, _ = strEqObj.(map[string]interface{})
	}
	if strEqMap == nil {
		strEqMap = make(map[string]interface{})
		condMap["StringEquals"] = strEqMap
	}

	if _, ok := strEqMap[audKey]; !ok {
		strEqMap[audKey] = []string{AWS_DEFAULT_AUDIENCE}
	}

	existingSubs := extractSubValues(strEqMap[subKey])
	for _, id := range existingSubs {
		if id == trustIdentity {
			return policyDocument, &ConditionExistError{}
		}
	}
	existingSubs = append(existingSubs, trustIdentity)
	strEqMap[subKey] = existingSubs

	statements[targetIdx] = stmt
	oldDoc["Statement"] = statements
	newPolicyDoc, err := json.Marshal(oldDoc)
	if err != nil {
		return "", err
	}
	return string(newPolicyDoc), nil
}

func removeServiceAccountInAssumeRolePolicy(policyDocument, serviceAccountNamespace, serviceAccountName string) (string, error) {
	var oldDoc MapOfInterfaces
	err := json.Unmarshal([]byte(policyDocument), &oldDoc)
	if err != nil {
		return "", err
	}
	var statements []MapOfInterfaces
	statementInBytes, err := json.Marshal(oldDoc["Statement"])
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(statementInBytes, &statements); err != nil {
		return "", err
	}

	targetIdx, oidcRoleArn, err := findKubeflowOIDCStatementIndex(statements, serviceAccountNamespace, serviceAccountName, true)
	if err != nil {
		return policyDocument, err
	}
	if targetIdx == -1 {
		return policyDocument, nil
	}

	issuerUrlWithProtocol := getIssuerUrlFromProviderArn(oidcRoleArn)
	subKey := fmt.Sprintf("%s:sub", issuerUrlWithProtocol)
	trustIdentity := fmt.Sprintf(AWS_TRUST_IDENTITY_SUBJECT, serviceAccountNamespace, serviceAccountName)

	stmt := statements[targetIdx]

	condObj, ok := stmt["Condition"]
	var condMap map[string]interface{}
	if ok && condObj != nil {
		condMap, _ = condObj.(map[string]interface{})
	}
	if condMap == nil {
		return policyDocument, nil
	}

	strEqObj, ok := condMap["StringEquals"]
	var strEqMap map[string]interface{}
	if ok && strEqObj != nil {
		strEqMap, _ = strEqObj.(map[string]interface{})
	}
	if strEqMap == nil {
		return policyDocument, nil
	}

	existingSubs := extractSubValues(strEqMap[subKey])
	var newSubs []string
	for _, id := range existingSubs {
		if id != trustIdentity {
			newSubs = append(newSubs, id)
		}
	}

	if len(newSubs) == 0 {
		delete(strEqMap, subKey)
	} else {
		strEqMap[subKey] = newSubs
	}

	statements[targetIdx] = stmt
	oldDoc["Statement"] = statements
	newPolicyDoc, err := json.Marshal(oldDoc)
	if err != nil {
		return "", err
	}
	return string(newPolicyDoc), nil
}

// getIssuerUrlFromProviderArn parse issuerUrl from Arn: arn:aws:iam::${accountId}:oidc-provider/${issuerUrl}
func getIssuerUrlFromProviderArn(arn string) string {
	return arn[strings.Index(arn, "/")+1:]
}

func getIAMRoleNameFromIAMRoleArn(arn string) string {
	return arn[strings.LastIndex(arn, "/")+1:]
}

// MakeAssumeRoleWithWebIdentityPolicyDocument constructs a trust policy for given a web identity provider with given conditions
func MakeAssumeRoleWithWebIdentityPolicyDocument(providerARN string, condition MapOfInterfaces) MapOfInterfaces {
	return MapOfInterfaces{
		"Effect": "Allow",
		"Action": "sts:AssumeRoleWithWebIdentity",
		"Principal": map[string]string{
			"Federated": providerARN,
		},
		"Condition": condition,
	}
}

// MakePolicyDocument constructs a policy with given statements
func MakePolicyDocument(statements ...MapOfInterfaces) MapOfInterfaces {
	return MapOfInterfaces{
		"Version":   "2012-10-17",
		"Statement": statements,
	}
}

func (aws *AwsIAMForServiceAccount) isAnnotateOnly() bool {
	return aws.AnnotateOnly
}

type (
	// MapOfInterfaces is an alias for map[string]interface{}
	MapOfInterfaces = map[string]interface{}
)

type IAMRole struct {
	AssumeRolePolicyDocument MapOfInterfaces `json:",omitempty"`
}

type ConditionExistError struct {
	msg string
}

func (e *ConditionExistError) Error() string {
	return e.msg
}
