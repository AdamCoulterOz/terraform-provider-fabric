// Copyright Microsoft Corporation 2026
// SPDX-License-Identifier: MPL-2.0

package workspacegit_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	azto "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	at "github.com/dcarbone/terraform-plugin-framework-utils/v3/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	fabcore "github.com/microsoft/fabric-sdk-go/fabric/core"

	"github.com/microsoft/terraform-provider-fabric/internal/common"
	"github.com/microsoft/terraform-provider-fabric/internal/testhelp"
	"github.com/microsoft/terraform-provider-fabric/internal/testhelp/fakes"
)

var testResourceItemFQN, testResourceItemHeader = testhelp.TFResource(common.ProviderTypeName, itemTypeInfo.Type, "test")

func TestUnit_WorkspaceGitResource_AzDO(t *testing.T) {
	gitConnection := NewRandomGitConnection(fabcore.GitProviderTypeAzureDevOps)
	gitCredentials := NewRandomGitCredentialsResponse(fabcore.GitCredentialsSourceAutomatic)
	gitProviderDetails := gitConnection.GitProviderDetails.GetGitProviderDetails()
	gitInit := NewRandomGitInitializeGitConnection()

	fakeServer := fakes.NewFakeServer()

	fakeServer.ServerFactory.Core.GitServer.GetConnection = fakeGitGetConnection(gitConnection)
	fakeServer.ServerFactory.Core.GitServer.GetMyGitCredentials = fakeGitGetMyGitCredentials(gitCredentials)
	fakeServer.ServerFactory.Core.GitServer.Connect = fakeGitConnect()
	fakeServer.ServerFactory.Core.GitServer.BeginInitializeConnection = fakeGitInitializeGitConnection(gitInit)
	fakeServer.ServerFactory.Core.GitServer.BeginCommitToGit = fakeGitCommitToGit()
	fakeServer.ServerFactory.Core.GitServer.BeginUpdateFromGit = fakeGitUpdateFromGit()
	fakeServer.ServerFactory.Core.GitServer.Disconnect = fakeGitDisconnect()

	testHelperGitProviderDetails := map[string]any{
		"git_provider_type": string(*gitProviderDetails.GitProviderType),
		"organization_name": "TestOrganization",
		"project_name":      "TestProject",
		"repository_name":   *gitProviderDetails.RepositoryName,
		"branch_name":       *gitProviderDetails.BranchName,
		"directory_name":    *gitProviderDetails.DirectoryName,
	}

	testCaseInvalidGitProviderType := testhelp.CopyMap(testHelperGitProviderDetails)
	testCaseInvalidGitProviderType["git_provider_type"] = "test1"

	testCaseInvalidDirectoryName := testhelp.CopyMap(testHelperGitProviderDetails)
	testCaseInvalidDirectoryName["directory_name"] = "test2"

	testCaseInvalidOwnerName := testhelp.CopyMap(testHelperGitProviderDetails)
	testCaseInvalidOwnerName["owner_name"] = "test3"

	testCaseMissingBranchName := testhelp.CopyMap(testHelperGitProviderDetails)
	delete(testCaseMissingBranchName, "branch_name")

	resource.ParallelTest(t, testhelp.NewTestUnitCase(t, &testResourceItemFQN, fakeServer.ServerFactory, nil, []resource.TestStep{
		// error - no attributes
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - no required git_credentials
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"git_provider_details":    testHelperGitProviderDetails,
					"initialization_strategy": "PreferWorkspace",
				},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - no required git_provider_details
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - no required initialization_strategy
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":         "00000000-0000-0000-0000-000000000000",
					"git_provider_details": testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - invalid initialization_strategy
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "test",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile(common.ErrorAttValueMatch),
		},
		// error - invalid git_provider_type
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseInvalidGitProviderType,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile(common.ErrorAttValueMatch),
		},
		// error - invalid directory_name
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseInvalidDirectoryName,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile(common.ErrorAttValueMatch),
		},
		// error - invalid owner_name
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseInvalidOwnerName,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile("Invalid configuration for attribute git_provider_details.owner_name"),
		},
		// error - missing branch_name
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseMissingBranchName,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile(`Incorrect attribute value type`),
		},
		// error - invalid git_credentials
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": "00000000-0000-0000-0000-000000000000",
					},
				},
			),
			ExpectError: regexp.MustCompile("Invalid configuration for attribute git_credentials.connection_id"),
		},
		// ok - PreferWorkspace
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPtr(testResourceItemFQN, "git_connection_state", (*string)(gitConnection.GitConnectionState)),
			),
		},
		// ok - PreferRemote
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferRemote",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPtr(testResourceItemFQN, "git_connection_state", (*string)(gitConnection.GitConnectionState)),
			),
		},
	}))
}

func TestUnit_WorkspaceGitResource_TargetCommit(t *testing.T) {
	gitConnection := NewRandomGitConnection(fabcore.GitProviderTypeAzureDevOps)
	gitCredentials := NewRandomGitCredentialsResponse(fabcore.GitCredentialsSourceAutomatic)
	gitProviderDetails := gitConnection.GitProviderDetails.GetGitProviderDetails()

	initialCommit := testhelp.RandomSHA1()
	targetCommit := testhelp.RandomSHA1()
	syncedCommit := initialCommit
	updateFromGitCalls := 0

	fakeServer := fakes.NewFakeServer()

	fakeServer.ServerFactory.Core.GitServer.GetConnection = func(_ context.Context, _ string, _ *fabcore.GitClientGetConnectionOptions) (resp azfake.Responder[fabcore.GitClientGetConnectionResponse], errResp azfake.ErrorResponder) {
		now := time.Now()
		gitConnection.GitSyncDetails.Head = &syncedCommit
		gitConnection.GitSyncDetails.LastSyncTime = &now

		resp = azfake.Responder[fabcore.GitClientGetConnectionResponse]{}
		resp.SetResponse(http.StatusOK, fabcore.GitClientGetConnectionResponse{GitConnection: gitConnection}, nil)

		return resp, errResp
	}
	fakeServer.ServerFactory.Core.GitServer.GetMyGitCredentials = fakeGitGetMyGitCredentials(gitCredentials)
	fakeServer.ServerFactory.Core.GitServer.Connect = fakeGitConnect()
	fakeServer.ServerFactory.Core.GitServer.BeginInitializeConnection = fakeGitInitializeGitConnection(fabcore.InitializeGitConnectionResponse{
		RequiredAction: azto.Ptr(fabcore.RequiredActionNone),
	})
	fakeServer.ServerFactory.Core.GitServer.BeginUpdateFromGit = func(_ context.Context, _ string, updateFromGitRequest fabcore.UpdateFromGitRequest, _ *fabcore.GitClientBeginUpdateFromGitOptions) (resp azfake.PollerResponder[fabcore.GitClientUpdateFromGitResponse], errResp azfake.ErrorResponder) {
		updateFromGitCalls++

		if updateFromGitRequest.RemoteCommitHash == nil || *updateFromGitRequest.RemoteCommitHash != targetCommit {
			t.Fatalf("expected update from Git to target commit %q, got %#v", targetCommit, updateFromGitRequest.RemoteCommitHash)
		}

		syncedCommit = *updateFromGitRequest.RemoteCommitHash

		resp = azfake.PollerResponder[fabcore.GitClientUpdateFromGitResponse]{}
		resp.SetTerminalResponse(http.StatusOK, fabcore.GitClientUpdateFromGitResponse{}, nil)

		return resp, errResp
	}
	fakeServer.ServerFactory.Core.GitServer.Disconnect = fakeGitDisconnect()

	testHelperGitProviderDetails := map[string]any{
		"git_provider_type": string(*gitProviderDetails.GitProviderType),
		"organization_name": "TestOrganization",
		"project_name":      "TestProject",
		"repository_name":   *gitProviderDetails.RepositoryName,
		"branch_name":       *gitProviderDetails.BranchName,
		"directory_name":    *gitProviderDetails.DirectoryName,
	}

	resource.ParallelTest(t, testhelp.NewTestUnitCase(t, &testResourceItemFQN, fakeServer.ServerFactory, nil, []resource.TestStep{
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"target_commit":           targetCommit,
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(testResourceItemFQN, "target_commit", targetCommit),
				resource.TestCheckResourceAttr(testResourceItemFQN, "git_sync_details.head", targetCommit),
				func(_ *terraform.State) error {
					if updateFromGitCalls != 1 {
						return fmt.Errorf("expected one update from Git call, got %d", updateFromGitCalls)
					}

					return nil
				},
			),
		},
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"target_commit":           targetCommit,
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
		},
	}))
}

func TestUnit_WorkspaceGitResource_TargetCommit_DriftAllowedWhenUnconfigured(t *testing.T) {
	gitConnection := NewRandomGitConnection(fabcore.GitProviderTypeAzureDevOps)
	gitCredentials := NewRandomGitCredentialsResponse(fabcore.GitCredentialsSourceAutomatic)
	gitProviderDetails := gitConnection.GitProviderDetails.GetGitProviderDetails()

	initialCommit := testhelp.RandomSHA1()
	driftedCommit := testhelp.RandomSHA1()
	syncedCommit := initialCommit
	updateFromGitCalls := 0

	fakeServer := fakes.NewFakeServer()

	fakeServer.ServerFactory.Core.GitServer.GetConnection = func(_ context.Context, _ string, _ *fabcore.GitClientGetConnectionOptions) (resp azfake.Responder[fabcore.GitClientGetConnectionResponse], errResp azfake.ErrorResponder) {
		now := time.Now()
		gitConnection.GitSyncDetails.Head = &syncedCommit
		gitConnection.GitSyncDetails.LastSyncTime = &now

		resp = azfake.Responder[fabcore.GitClientGetConnectionResponse]{}
		resp.SetResponse(http.StatusOK, fabcore.GitClientGetConnectionResponse{GitConnection: gitConnection}, nil)

		return resp, errResp
	}
	fakeServer.ServerFactory.Core.GitServer.GetMyGitCredentials = fakeGitGetMyGitCredentials(gitCredentials)
	fakeServer.ServerFactory.Core.GitServer.Connect = fakeGitConnect()
	fakeServer.ServerFactory.Core.GitServer.BeginInitializeConnection = fakeGitInitializeGitConnection(fabcore.InitializeGitConnectionResponse{
		RequiredAction: azto.Ptr(fabcore.RequiredActionNone),
	})
	fakeServer.ServerFactory.Core.GitServer.BeginUpdateFromGit = func(_ context.Context, _ string, _ fabcore.UpdateFromGitRequest, _ *fabcore.GitClientBeginUpdateFromGitOptions) (resp azfake.PollerResponder[fabcore.GitClientUpdateFromGitResponse], errResp azfake.ErrorResponder) {
		updateFromGitCalls++

		resp = azfake.PollerResponder[fabcore.GitClientUpdateFromGitResponse]{}
		resp.SetTerminalResponse(http.StatusOK, fabcore.GitClientUpdateFromGitResponse{}, nil)

		return resp, errResp
	}
	fakeServer.ServerFactory.Core.GitServer.Disconnect = fakeGitDisconnect()

	testHelperGitProviderDetails := map[string]any{
		"git_provider_type": string(*gitProviderDetails.GitProviderType),
		"organization_name": "TestOrganization",
		"project_name":      "TestProject",
		"repository_name":   *gitProviderDetails.RepositoryName,
		"branch_name":       *gitProviderDetails.BranchName,
		"directory_name":    *gitProviderDetails.DirectoryName,
	}

	config := at.CompileConfig(
		testResourceItemHeader,
		map[string]any{
			"workspace_id":            "00000000-0000-0000-0000-000000000000",
			"initialization_strategy": "PreferWorkspace",
			"git_provider_details":    testHelperGitProviderDetails,
			"git_credentials": map[string]any{
				"source": string(fabcore.GitCredentialsSourceAutomatic),
			},
		},
	)

	resource.ParallelTest(t, testhelp.NewTestUnitCase(t, &testResourceItemFQN, fakeServer.ServerFactory, nil, []resource.TestStep{
		{
			ResourceName: testResourceItemFQN,
			Config:       config,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(testResourceItemFQN, "target_commit", initialCommit),
				func(_ *terraform.State) error {
					syncedCommit = driftedCommit

					return nil
				},
			),
		},
		{
			ResourceName: testResourceItemFQN,
			Config:       config,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectEmptyPlan(),
				},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(testResourceItemFQN, "target_commit", driftedCommit),
				func(_ *terraform.State) error {
					if updateFromGitCalls != 0 {
						return fmt.Errorf("expected no update from Git calls, got %d", updateFromGitCalls)
					}

					return nil
				},
			),
		},
	}))
}

func TestAcc_WorkspaceGitResource_AzDO_Automatic(t *testing.T) {
	if testhelp.ShouldSkipTest(t) {
		t.Skip("No SPN support")
	}

	capacity := testhelp.WellKnown()["Capacity"].(map[string]any)
	capacityID := capacity["id"].(string)

	doPlatform := testhelp.WellKnown()["AzDO"].(map[string]any)
	azdoOrganization := doPlatform["organizationName"].(string)
	azdoProject := doPlatform["projectName"].(string)
	azdoRepository := doPlatform["repositoryName"].(string)
	adoConnectionID := doPlatform["connectionId"].(string)

	workspaceResourceHCL, workspaceResourceFQN := testhelp.TestAccWorkspaceResource(t, capacityID)

	resource.Test(t, testhelp.NewTestAccCase(t, &testResourceItemFQN, nil, []resource.TestStep{
		// Create and Read
		{
			ResourceName: testResourceItemFQN,
			Config: at.JoinConfigs(
				workspaceResourceHCL,
				at.CompileConfig(
					testResourceItemHeader,
					map[string]any{
						"workspace_id":            testhelp.RefByFQN(workspaceResourceFQN, "id"),
						"initialization_strategy": "PreferWorkspace",
						"git_provider_details": map[string]any{
							"git_provider_type": "AzureDevOps",
							"organization_name": azdoOrganization,
							"project_name":      azdoProject,
							"repository_name":   azdoRepository,
							"branch_name":       "main",
							"directory_name":    "/",
						},
						"git_credentials": map[string]any{
							"source": string(fabcore.GitCredentialsSourceAutomatic),
						},
					},
				)),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet(testResourceItemFQN, "git_sync_details.head"),
				resource.TestCheckResourceAttr(testResourceItemFQN, "git_connection_state", string(fabcore.GitConnectionStateConnectedAndInitialized)),
			),
		},
		// Update git_credentials to ConfiguredConnection - this should update in-place only.
		{
			ResourceName: testResourceItemFQN,
			Config: at.JoinConfigs(
				workspaceResourceHCL,
				at.CompileConfig(
					testResourceItemHeader,
					map[string]any{
						"workspace_id":            testhelp.RefByFQN(workspaceResourceFQN, "id"),
						"initialization_strategy": "PreferWorkspace",
						"git_provider_details": map[string]any{
							"git_provider_type": "AzureDevOps",
							"organization_name": azdoOrganization,
							"project_name":      azdoProject,
							"repository_name":   azdoRepository,
							"branch_name":       "main",
							"directory_name":    "/",
						},
						"git_credentials": map[string]any{
							"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
							"connection_id": adoConnectionID,
						},
					},
				)),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(testResourceItemFQN, plancheck.ResourceActionUpdate),
				},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet(testResourceItemFQN, "git_sync_details.head"),
				resource.TestCheckResourceAttr(testResourceItemFQN, "git_connection_state", string(fabcore.GitConnectionStateConnectedAndInitialized)),
			),
		},
	}))
}

func TestAcc_WorkspaceGitResource_AzDO_ConfiguredCredentials(t *testing.T) {
	capacity := testhelp.WellKnown()["Capacity"].(map[string]any)
	capacityID := capacity["id"].(string)

	doPlatform := testhelp.WellKnown()["AzDO"].(map[string]any)
	azdoOrganization := doPlatform["organizationName"].(string)
	azdoProject := doPlatform["projectName"].(string)
	azdoRepository := doPlatform["repositoryName"].(string)
	adoConnectionID := doPlatform["connectionId"].(string)

	workspaceResourceHCL, workspaceResourceFQN := testhelp.TestAccWorkspaceResource(t, capacityID)

	resource.Test(t, testhelp.NewTestAccCase(t, &testResourceItemFQN, nil, []resource.TestStep{
		// Create and Read
		{
			ResourceName: testResourceItemFQN,
			Config: at.JoinConfigs(
				workspaceResourceHCL,
				at.CompileConfig(
					testResourceItemHeader,
					map[string]any{
						"workspace_id":            testhelp.RefByFQN(workspaceResourceFQN, "id"),
						"initialization_strategy": "PreferWorkspace",
						"git_provider_details": map[string]any{
							"git_provider_type": "AzureDevOps",
							"organization_name": azdoOrganization,
							"project_name":      azdoProject,
							"repository_name":   azdoRepository,
							"branch_name":       "main",
							"directory_name":    "/",
						},
						"git_credentials": map[string]any{
							"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
							"connection_id": adoConnectionID,
						},
					},
				)),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet(testResourceItemFQN, "git_sync_details.head"),
				resource.TestCheckResourceAttr(testResourceItemFQN, "git_connection_state", string(fabcore.GitConnectionStateConnectedAndInitialized)),
			),
		},
	}))
}

func TestUnit_WorkspaceGitResource_GitHub(t *testing.T) {
	gitConnection := NewRandomGitConnection(fabcore.GitProviderTypeGitHub)
	gitCredentials := NewRandomGitCredentialsResponse(fabcore.GitCredentialsSourceConfiguredConnection)
	gitProviderDetails := gitConnection.GitProviderDetails.GetGitProviderDetails()
	gitInit := NewRandomGitInitializeGitConnection()

	fakeServer := fakes.NewFakeServer()

	fakeServer.ServerFactory.Core.GitServer.GetConnection = fakeGitGetConnection(gitConnection)
	fakeServer.ServerFactory.Core.GitServer.GetMyGitCredentials = fakeGitGetMyGitCredentials(gitCredentials)
	fakeServer.ServerFactory.Core.GitServer.Connect = fakeGitConnect()
	fakeServer.ServerFactory.Core.GitServer.BeginInitializeConnection = fakeGitInitializeGitConnection(gitInit)
	fakeServer.ServerFactory.Core.GitServer.BeginCommitToGit = fakeGitCommitToGit()
	fakeServer.ServerFactory.Core.GitServer.BeginUpdateFromGit = fakeGitUpdateFromGit()
	fakeServer.ServerFactory.Core.GitServer.Disconnect = fakeGitDisconnect()

	gitCredentialsResponse := gitCredentials.GitCredentialsConfigurationResponseClassification.(*fabcore.ConfiguredConnectionGitCredentialsResponse)

	testHelperGitProviderDetails := map[string]any{
		"git_provider_type": string(*gitProviderDetails.GitProviderType),
		"owner_name":        "TestOwner",
		"repository_name":   *gitProviderDetails.RepositoryName,
		"branch_name":       *gitProviderDetails.BranchName,
		"directory_name":    *gitProviderDetails.DirectoryName,
	}

	testCaseInvalidGitProviderType := testhelp.CopyMap(testHelperGitProviderDetails)
	testCaseInvalidGitProviderType["git_provider_type"] = "test1"

	testCaseInvalidDirectoryName := testhelp.CopyMap(testHelperGitProviderDetails)
	testCaseInvalidDirectoryName["directory_name"] = "test2"

	testCaseInvalidOrganizationName := testhelp.CopyMap(testHelperGitProviderDetails)
	testCaseInvalidOrganizationName["organization_name"] = "test3"

	testCaseMissingBranchName := testhelp.CopyMap(testHelperGitProviderDetails)
	delete(testCaseMissingBranchName, "branch_name")

	resource.ParallelTest(t, testhelp.NewTestUnitCase(t, &testResourceItemFQN, fakeServer.ServerFactory, nil, []resource.TestStep{
		// error - no attributes
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - no required git_provider_details
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - no required git_credentials
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"git_provider_details":    testHelperGitProviderDetails,
					"initialization_strategy": "PreferWorkspace",
				},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - no required git_credentials.connection_id
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"git_provider_details":    testHelperGitProviderDetails,
					"initialization_strategy": "PreferWorkspace",
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceConfiguredConnection),
					},
				},
			),
			ExpectError: regexp.MustCompile(`Invalid configuration for attribute git_credentials.connection_id`),
		},
		// error - invalid git_credentials.source
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"git_provider_details":    testHelperGitProviderDetails,
					"initialization_strategy": "PreferWorkspace",
					"git_credentials": map[string]any{
						"source": string(fabcore.GitCredentialsSourceAutomatic),
					},
				},
			),
			ExpectError: regexp.MustCompile(`Invalid configuration for attribute git_credentials.source`),
		},
		// error - no required initialization_strategy
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":         "00000000-0000-0000-0000-000000000000",
					"git_provider_details": testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile(`Missing required argument`),
		},
		// error - invalid initialization_strategy
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "test",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile(common.ErrorAttValueMatch),
		},
		// error - invalid git_provider_type
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseInvalidGitProviderType,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile(common.ErrorAttValueMatch),
		},
		// error - invalid directory_name
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseInvalidDirectoryName,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile(common.ErrorAttValueMatch),
		},
		// error - invalid owner_name
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseInvalidOrganizationName,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile("Invalid configuration for attribute git_provider_details.organization_name"),
		},
		// error - missing branch_name
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testCaseMissingBranchName,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile(`Incorrect attribute value type`),
		},
		// error - invalid git_credentials source
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceAutomatic),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			ExpectError: regexp.MustCompile(`Invalid configuration for attribute git_credentials.source`),
		},
		// ok - PreferWorkspace
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPtr(testResourceItemFQN, "git_connection_state", (*string)(gitConnection.GitConnectionState)),
			),
		},
		// ok - PreferRemote
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferRemote",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPtr(testResourceItemFQN, "git_connection_state", (*string)(gitConnection.GitConnectionState)),
			),
		},
		// ok - optional git_credentials source
		{
			ResourceName: testResourceItemFQN,
			Config: at.CompileConfig(
				testResourceItemHeader,
				map[string]any{
					"workspace_id":            "00000000-0000-0000-0000-000000000000",
					"initialization_strategy": "PreferWorkspace",
					"git_provider_details":    testHelperGitProviderDetails,
					"git_credentials": map[string]any{
						"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
						"connection_id": *gitCredentialsResponse.ConnectionID,
					},
				},
			),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrPtr(testResourceItemFQN, "git_connection_state", (*string)(gitConnection.GitConnectionState)),
			),
		},
	}))
}

func TestAcc_WorkspaceGitResource_GitHub_ConfiguredCredentials(t *testing.T) {
	capacity := testhelp.WellKnown()["Capacity"].(map[string]any)
	capacityID := capacity["id"].(string)

	doPlatform := testhelp.WellKnown()["GitHub"].(map[string]any)
	ghOwner := doPlatform["ownerName"].(string)
	ghRepository := doPlatform["repositoryName"].(string)
	ghConnectionID := doPlatform["connectionId"].(string)

	workspaceResourceHCL, workspaceResourceFQN := testhelp.TestAccWorkspaceResource(t, capacityID)

	resource.Test(t, testhelp.NewTestAccCase(t, &testResourceItemFQN, nil, []resource.TestStep{
		// Create and Read - with allow_override_items
		{
			ResourceName: testResourceItemFQN,
			Config: at.JoinConfigs(
				workspaceResourceHCL,
				at.CompileConfig(
					testResourceItemHeader,
					map[string]any{
						"workspace_id":            testhelp.RefByFQN(workspaceResourceFQN, "id"),
						"initialization_strategy": "PreferWorkspace",
						"options": map[string]any{
							"allow_override_items": true,
						},
						"git_provider_details": map[string]any{
							"git_provider_type": "GitHub",
							"owner_name":        ghOwner,
							"repository_name":   ghRepository,
							"branch_name":       "main",
							"directory_name":    "/",
						},
						"git_credentials": map[string]any{
							"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
							"connection_id": ghConnectionID,
						},
					},
				)),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttrSet(testResourceItemFQN, "git_sync_details.head"),
				resource.TestCheckResourceAttr(testResourceItemFQN, "git_connection_state", string(fabcore.GitConnectionStateConnectedAndInitialized)),
			),
		},
		// Recreate and Read - without allow_override_items
		{
			ResourceName: testResourceItemFQN,
			Config: at.JoinConfigs(
				workspaceResourceHCL,
				at.CompileConfig(
					testResourceItemHeader,
					map[string]any{
						"workspace_id":            testhelp.RefByFQN(workspaceResourceFQN, "id"),
						"initialization_strategy": "PreferWorkspace",
						"git_provider_details": map[string]any{
							"git_provider_type": "GitHub",
							"owner_name":        ghOwner,
							"repository_name":   ghRepository,
							"branch_name":       "main",
							"directory_name":    "/",
						},
						"git_credentials": map[string]any{
							"source":        string(fabcore.GitCredentialsSourceConfiguredConnection),
							"connection_id": ghConnectionID,
						},
					},
				)),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckNoResourceAttr(testResourceItemFQN, "options"),
				resource.TestCheckResourceAttrSet(testResourceItemFQN, "git_sync_details.head"),
				resource.TestCheckResourceAttr(testResourceItemFQN, "git_connection_state", string(fabcore.GitConnectionStateConnectedAndInitialized)),
			),
		},
	},
	))
}
