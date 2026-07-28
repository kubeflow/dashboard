import 'jasmine';
import express, {Request, Response} from 'express';

import {attachUser} from './attach_user_middleware';
import {DefaultApi} from './clients/profile_controller';
import {KubernetesService} from './k8s_service';
import {
    WorkgroupApi,
} from './api_workgroup';
import {sendTestRequest} from './test_resources';

describe('Workgroup API', () => {
    const header = {
        goog: 'X-Goog-Authenticated-User-Email',
        other: 'Other-Header',
    };
    const prefix = {
        goog: 'accounts.google.com:',
        other: 'other.foo.bar:',
    };
    const attachUserGCPMiddleware = attachUser(header.goog, prefix.goog);
    const attachUserOtherIAPMiddleware = attachUser(header.other, prefix.other);
    const registrationFlowAllowed = true;
    let mockK8sService: jasmine.SpyObj<KubernetesService>;
    let mockProfilesService: jasmine.SpyObj<DefaultApi>;
    let testApp: express.Application;
    let port: number;
    const newAPI = () => new WorkgroupApi(
        mockProfilesService,
        mockK8sService,
        registrationFlowAllowed,
    );

    describe('Environment Information', () => {
        let url: string;
        beforeEach(() => {
            mockK8sService = jasmine.createSpyObj<KubernetesService>([
                'getPlatformInfo',
                'getNamespaces',
            ]);
            mockK8sService.getPlatformInfo.and.returnValue(Promise.resolve({
                provider: 'onprem',
                providerName: 'onprem',
                kubeflowVersion: '1.0.0',
            }));
            mockProfilesService = jasmine.createSpyObj<DefaultApi>(
                ['readBindings', 'v1RoleClusteradminGet']);

            mockProfilesService.readBindings.withArgs()
                .and.returnValue(Promise.resolve({
                    response: null,
                    body: {
                        bindings: [
                            {
                                user: {kind: 'User', name: 'anyone@kubeflow.org'},
                                referredNamespace: 'default',
                                roleRef: {kind: 'ClusterRole', name: 'admin'},
                            },
                            {
                                user: {kind: 'User', name: 'user1@kubeflow.org'},
                                referredNamespace: 'default',
                                roleRef: {kind: 'ClusterRole', name: 'edit'},
                            },
                            {
                                user: {kind: 'User', name: 'user1@kubeflow.org'},
                                referredNamespace: 'kubeflow',
                                roleRef: {kind: 'ClusterRole', name: 'admin'},
                            },
                        ]
                    },
                }));

            testApp = express();
            testApp.use(express.json());
            testApp.use(attachUserGCPMiddleware);
            testApp.use('/api/workgroup', newAPI().routes());
            const addressInfo = testApp.listen(0).address();
            if (typeof addressInfo === 'string') {
                throw new Error(
                    'Unable to determine system-assigned port for test API server');
            }
            port = addressInfo.port;
            url = `http://localhost:${port}/api/workgroup/env-info`;
        });

        it('Should retrieve information for a non-identity aware cluster',
            async () => {
                const expectedResponse = {
                    platform: {
                        provider: 'onprem',
                        providerName: 'onprem',
                        kubeflowVersion: '1.0.0',
                    },
                    user: 'anonymous@kubeflow.org',
                    isClusterAdmin: true,
                    namespaces: [
                        {
                            user: 'anonymous@kubeflow.org',
                            namespace: 'default',
                            role: 'contributor',
                        },
                        {
                            user: 'anonymous@kubeflow.org',
                            namespace: 'kubeflow',
                            role: 'contributor',
                        },
                    ],
                };

                let response = await sendTestRequest(url);
                expect(response).toEqual(expectedResponse);
                expect(mockK8sService.getPlatformInfo).toHaveBeenCalled();

                // Second call should use cached platform information
                response = await sendTestRequest(url);
                expect(response).toEqual(expectedResponse);
                expect(mockK8sService.getPlatformInfo.calls.count()).toBe(1);
                expect(mockProfilesService.readBindings).toHaveBeenCalled();
                expect(mockProfilesService.v1RoleClusteradminGet)
                    .not.toHaveBeenCalled();
            });

        it('Should retrieve information for an identity aware cluster',
            async () => {
                mockProfilesService.v1RoleClusteradminGet
                    .withArgs('test@testdomain.com')
                    .and.returnValue(Promise.resolve({response: null, body: false}));
                mockProfilesService.readBindings.withArgs('test@testdomain.com')
                    .and.returnValue(Promise.resolve({
                        response: null,
                        body: {
                            bindings: [{
                                user: {kind: 'user', name: 'test@testdomain.com'},
                                referredNamespace: 'test',
                                roleRef: {apiGroup: '', kind: 'ClusterRole', name: 'edit'}
                            }]
                        },
                    }));

                const headers = {
                    [header.goog]: `${prefix.goog}test@testdomain.com`,
                };
                const expectedResponse = {
                    platform: {
                        provider: 'onprem',
                        providerName: 'onprem',
                        kubeflowVersion: '1.0.0',
                    },
                    user: 'test@testdomain.com',
                    isClusterAdmin: false,
                    namespaces: [
                        {
                            user: 'test@testdomain.com',
                            namespace: 'test',
                            role: 'contributor',
                        },
                    ],
                };

                const response = await sendTestRequest(url, headers);
                expect(response).toEqual(expectedResponse);
                expect(mockK8sService.getNamespaces).not.toHaveBeenCalled();
                expect(mockK8sService.getPlatformInfo).toHaveBeenCalled();
                expect(mockProfilesService.readBindings)
                    .toHaveBeenCalledWith('test@testdomain.com');
                expect(mockProfilesService.v1RoleClusteradminGet)
                    .toHaveBeenCalledWith('test@testdomain.com');
            });

        it('Returns an error status if the Profiles service fails', async () => {
            mockProfilesService.v1RoleClusteradminGet.withArgs('test@testdomain.com')
                .and.callFake(
                    () => Promise.reject(
                        {response: {statusCode: 400}, body: 'A bad thing happened'}));
            mockProfilesService.readBindings.withArgs('test@testdomain.com')
                .and.returnValue(Promise.resolve({
                    response: null,
                    body: {
                        bindings: [{
                            user: {kind: 'user', name: 'test@testdomain.com'},
                            referredNamespace: 'test',
                            roleRef: {apiGroup: '', kind: 'ClusterRole', name: 'edit'}
                        }]
                    },
                }));

            const headers = {
                [header.goog]: `${prefix.goog}test@testdomain.com`,
            };
            const response = await sendTestRequest(url, headers, 400);
            expect(response).toEqual({error: 'A bad thing happened'});
            expect(mockK8sService.getNamespaces).not.toHaveBeenCalled();
            expect(mockK8sService.getPlatformInfo).toHaveBeenCalled();
            expect(mockProfilesService.readBindings)
                .toHaveBeenCalledWith('test@testdomain.com');
            expect(mockProfilesService.v1RoleClusteradminGet)
                .toHaveBeenCalledWith('test@testdomain.com');
        });
    });

    describe('Has Workgroup', () => {
        let url: string;
        beforeEach(() => {
            mockProfilesService = jasmine.createSpyObj<DefaultApi>(
                ['readBindings', 'v1RoleClusteradminGet']);

            testApp = express();
            testApp.use(express.json());
            testApp.use(attachUserGCPMiddleware);
            testApp.use('/api/workgroup', newAPI().routes());
            const addressInfo = testApp.listen(0).address();
            if (typeof addressInfo === 'string') {
                throw new Error(
                    'Unable to determine system-assigned port for test API server');
            }
            port = addressInfo.port;
            url = `http://localhost:${port}/api/workgroup/exists`;
        });

        it('Should return for a non-identity aware cluster', async () => {
            mockProfilesService.readBindings.withArgs()
                .and.returnValue(Promise.resolve({
                    response: null,
                    body: {
                        bindings: []
                    },
                }));
            const expectedResponse = {hasAuth: false, hasWorkgroup: false, 
                user: 'anonymous', registrationFlowAllowed: true};

            const response = await sendTestRequest(url);
            expect(response).toEqual(expectedResponse);
            expect(mockProfilesService.v1RoleClusteradminGet).not.toHaveBeenCalled();
            expect(mockProfilesService.readBindings).toHaveBeenCalledWith();
        });

        it('Should return for an identity aware cluster with a Workgroup',
            async () => {
                mockProfilesService.v1RoleClusteradminGet
                    .withArgs('test@testdomain.com')
                    .and.returnValue(Promise.resolve({response: null, body: false}));
                mockProfilesService.readBindings.withArgs('test@testdomain.com')
                    .and.returnValue(Promise.resolve({
                        response: null,
                        body: {
                            bindings: [{
                                user: {kind: 'user', name: 'test@testdomain.com'},
                                referredNamespace: 'test',
                                roleRef: {apiGroup: '', kind: 'ClusterRole', name: 'admin'}
                            }]
                        },
                    }));

                const expectedResponse = {hasAuth: true, hasWorkgroup: true, 
                    user: 'test', registrationFlowAllowed: true};

                const headers = {
                    [header.goog]: `${prefix.goog}test@testdomain.com`,
                };
                const response = await sendTestRequest(url, headers);
                expect(response).toEqual(expectedResponse);
                expect(mockProfilesService.readBindings)
                    .toHaveBeenCalledWith('test@testdomain.com');
                expect(mockProfilesService.v1RoleClusteradminGet)
                    .toHaveBeenCalledWith('test@testdomain.com');
            });

        it('Should return for an identity aware cluster without a Workgroup', async () => {
            mockProfilesService.v1RoleClusteradminGet
                .withArgs('test@testdomain.com')
                .and.returnValue(Promise.resolve({response: null, body: false}));
            mockProfilesService.readBindings.withArgs('test@testdomain.com')
                .and.returnValue(Promise.resolve({
                    response: null,
                    body: {bindings: []},
                }));

            const expectedResponse = {hasAuth: true, hasWorkgroup: false, 
                user: 'test', registrationFlowAllowed: true};

            const headers = {
                [header.goog]: `${prefix.goog}test@testdomain.com`,
            };
            const response = await sendTestRequest(url, headers);
            expect(response).toEqual(expectedResponse);
            expect(mockProfilesService.readBindings)
                .toHaveBeenCalledWith('test@testdomain.com');
            expect(mockProfilesService.v1RoleClusteradminGet)
                .toHaveBeenCalledWith('test@testdomain.com');
        });
    });

    describe('Create Workgroup', () => {
        let url: string;

        beforeEach(() => {
            mockProfilesService = jasmine.createSpyObj<DefaultApi>(['createProfile']);

            testApp = express();
            testApp.use(express.json());
            testApp.use(attachUserGCPMiddleware);
            testApp.use('/api/workgroup', newAPI().routes());
            const addressInfo = testApp.listen(0).address();
            if (typeof addressInfo === 'string') {
                throw new Error(
                    'Unable to determine system-assigned port for test API server');
            }
            port = addressInfo.port;
            url = `http://localhost:${port}/api/workgroup/create`;
        });

        it('Should work for a non-identity aware cluster', async () => {
            const response = await sendTestRequest(url, null, 200, 'post');
            expect(response).toEqual({message: 'Created namespace anonymous'});
            expect(mockProfilesService.createProfile).toHaveBeenCalledWith({
                metadata: {
                    name: 'anonymous',
                },
                spec: {
                    owner: {
                        kind: 'User',
                        name: 'anonymous@kubeflow.org',
                    }
                },
            });
        });

        it('Should use user identity if no body is provided', async () => {
            const headers = {
                [header.goog]: `${prefix.goog}test@testdomain.com`,
            };
            const response = await sendTestRequest(url, headers, 200, 'post');
            expect(response).toEqual({message: 'Created namespace test'});
            expect(mockProfilesService.createProfile).toHaveBeenCalledWith({
                metadata: {
                    name: 'test',
                },
                spec: {
                    owner: {
                        kind: 'User',
                        name: 'test@testdomain.com',
                    }
                },
            });
        });

        it('Should use post body when provided', async () => {
            const headers = {
                [header.goog]: `${prefix.goog}test@testdomain.com`,
                'content-type': 'application/json',
            };
            const response = await sendTestRequest(
                url, headers, 200, 'post',
                {namespace: 'a_different_namespace', user: 'another_user@foo.bar'});
            expect(response).toEqual({message: 'Created namespace a_different_namespace'});
            expect(mockProfilesService.createProfile).toHaveBeenCalledWith({
                metadata: {
                    name: 'a_different_namespace',
                },
                spec: {
                    owner: {
                        kind: 'User',
                        name: 'another_user@foo.bar',
                    }
                },
            });
        });

        it('Returns an error status if the Profiles service fails', async () => {
            mockProfilesService.createProfile
                .withArgs({
                    metadata: {
                        name: 'test',
                    },
                    spec: {
                        owner: {
                            kind: 'User',
                            name: 'test@testdomain.com',
                        }
                    },
                })
                .and.callFake(() => Promise.reject({response: {statusCode: 405}}));

            const headers = {
                [header.goog]: `${prefix.goog}test@testdomain.com`,
            };
            const response = await sendTestRequest(url, headers, 405, 'post');
            expect(response).toEqual({error: 'Unexpected error creating profile'});
        });
    });
    describe('Add / Remove Contributor', () => {
        type RouteTypes = 'add' | 'add-viewer' | 'remove';
        let url: (type: RouteTypes) => string;
        const requestBody = {contributor: 'apverma@google.com'};
        const headers = {
            'content-type': 'application/json',
            [header.goog]: `${prefix.goog}test@testdomain.com`,
        };
        const existingContributors = [
            {user: 'apverma@google.com', role: 'contributor'},
            {user: 'viewer@example.com', role: 'viewer'},
        ];

        const buildApi = (contributors = []) => {
            mockProfilesService = jasmine.createSpyObj<DefaultApi>(['createBinding', 'deleteBinding']);
            const api = newAPI();
            api.getContributors = async () => contributors;
            testApp = express();
            testApp.use(express.json());
            testApp.use(attachUserGCPMiddleware);
            testApp.use('/api/workgroup', api.routes());
            const addressInfo = testApp.listen(0).address();
            if (typeof addressInfo === 'string') {
                throw new Error(
                    'Unable to determine system-assigned port for test API server');
            }
            port = addressInfo.port;
        };

        beforeEach(() => {
            buildApi();
            url = (type: RouteTypes) =>
                `http://localhost:${port}/api/workgroup/${type}-contributor/apverma`;
        });
        it('Should should show error if user auth status is not detected', async () => {
            const response = await sendTestRequest(url('add'), null, 405, 'post', requestBody);
            expect(response).toEqual({error: `Unable to ascertain user identity from request, cannot access route.`});
            expect(mockProfilesService.createBinding).not.toHaveBeenCalled();
        });
        it('Should error on missing contributor', async () => {
            const [rAdd, rViewer, rRemove] = await Promise.all([
                sendTestRequest(url('add'), headers, 400, 'post'),
                sendTestRequest(`http://localhost:${port}/api/workgroup/add-viewer/apverma`, headers, 400, 'post'),
                sendTestRequest(url('remove'), headers, 400, 'delete'),
            ]);
            [rAdd, rViewer, rRemove].forEach(response => {
                expect(response).toEqual({error: `Missing contributor field.`});
            });
            expect(mockProfilesService.createBinding).not.toHaveBeenCalled();
        });
        it('Should error on invalid email for contrib', async () => {
            const response = await sendTestRequest(url('add'), headers, 400, 'post', {
                contributor: 'apverma'
            });
            expect(response).toEqual({error: `Contributor doesn't look like a valid email address`});
            expect(mockProfilesService.createBinding).not.toHaveBeenCalled();
        });
        it('Should successfully add a new contributor via add-contributor endpoint', async () => {
            const response = await sendTestRequest(url('add'), headers, 200, 'post', requestBody);
            expect(response).toEqual([]);
            expect(mockProfilesService.createBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'apverma@google.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'edit'},
            }, jasmine.anything());
            expect(mockProfilesService.deleteBinding).not.toHaveBeenCalled();
        });
        it('Should successfully add a new viewer via add-viewer endpoint', async () => {
            const viewerUrl = `http://localhost:${port}/api/workgroup/add-viewer/apverma`;
            const response = await sendTestRequest(viewerUrl, headers, 200, 'post', requestBody);
            expect(response).toEqual([]);
            expect(mockProfilesService.createBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'apverma@google.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'view'},
            }, jasmine.anything());
            expect(mockProfilesService.deleteBinding).not.toHaveBeenCalled();
        });
        it('Should bubble kfam error when adding a user with the same role they already have', async () => {
            buildApi(existingContributors);
            mockProfilesService.createBinding.and.rejectWith({
                response: {statusCode: 409, statusMessage: 'Conflict'},
                body: 'rolebindings.rbac.authorization.k8s.io "user-apverma-clusterrole-edit" already exists',
            });
            const response = await sendTestRequest(
                `http://localhost:${port}/api/workgroup/add-contributor/apverma`,
                headers, 409, 'post', requestBody,
            );
            expect(mockProfilesService.deleteBinding).not.toHaveBeenCalled();
            expect(mockProfilesService.createBinding).toHaveBeenCalled();
            expect(response.error).toContain('already exists');
        });
        it('Should remove old binding and create new one when upgrading role', async () => {
            buildApi(existingContributors);
            // viewer@example.com is currently a viewer — upgrade to contributor
            const response = await sendTestRequest(
                `http://localhost:${port}/api/workgroup/add-contributor/apverma`,
                headers, 200, 'post', {contributor: 'viewer@example.com'},
            );
            expect(response).toEqual(existingContributors);
            expect(mockProfilesService.deleteBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'viewer@example.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'view'},
            }, jasmine.anything());
            expect(mockProfilesService.createBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'viewer@example.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'edit'},
            }, jasmine.anything());
        });
        it('Should remove old binding and create new one when downgrading role', async () => {
            buildApi(existingContributors);
            // apverma@google.com is currently a contributor — downgrade to viewer
            const response = await sendTestRequest(
                `http://localhost:${port}/api/workgroup/add-viewer/apverma`,
                headers, 200, 'post', requestBody,
            );
            expect(response).toEqual(existingContributors);
            expect(mockProfilesService.deleteBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'apverma@google.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'edit'},
            }, jasmine.anything());
            expect(mockProfilesService.createBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'apverma@google.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'view'},
            }, jasmine.anything());
        });
        it('Should error with cleanup message when old binding delete fails during role change', async () => {
            buildApi(existingContributors);
            // viewer@example.com is currently a viewer — upgrade to contributor
            // createBinding succeeds, but deleteBinding (cleanup of old role) fails
            mockProfilesService.deleteBinding.and.rejectWith({
                response: {statusCode: 500, statusMessage: 'Internal Server Error'},
            });
            const response = await sendTestRequest(
                `http://localhost:${port}/api/workgroup/add-contributor/apverma`,
                headers, 500, 'post', {contributor: 'viewer@example.com'},
            );
            expect(response.error).toContain('Role updated but failed to remove existing assignment');
            expect(response.error).toContain('viewer@example.com');
            // new binding was created before the failed cleanup
            expect(mockProfilesService.createBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'viewer@example.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'edit'},
            }, jasmine.anything());
            expect(mockProfilesService.deleteBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'viewer@example.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'view'},
            }, jasmine.anything());
        });
        it('Should error when removing a user not in the namespace', async () => {
            const response = await sendTestRequest(url('remove'), {...headers, 'Transfer-Encoding': 'chunked'}, 400, 'delete', {
                contributor: 'unknown@google.com',
            });
            expect(response).toEqual({error: `unknown@google.com is not a contributor of apverma`});
            expect(mockProfilesService.deleteBinding).not.toHaveBeenCalled();
        });
        it('Should successfully remove a contributor', async () => {
            buildApi(existingContributors);
            const response = await sendTestRequest(url('remove'), {...headers, 'Transfer-Encoding': 'chunked'}, 200, 'delete', requestBody);
            expect(response).toEqual(existingContributors);
            expect(mockProfilesService.createBinding).not.toHaveBeenCalled();
            expect(mockProfilesService.deleteBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'apverma@google.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'edit'},
            }, jasmine.anything());
        });
        it('Should successfully remove a viewer', async () => {
            buildApi(existingContributors);
            const removeViewerUrl =
                `http://localhost:${port}/api/workgroup/remove-viewer/apverma`;
            const response = await sendTestRequest(removeViewerUrl, {...headers, 'Transfer-Encoding': 'chunked'}, 200, 'delete', {
                contributor: 'viewer@example.com',
            });
            expect(response).toEqual(existingContributors);
            expect(mockProfilesService.createBinding).not.toHaveBeenCalled();
            expect(mockProfilesService.deleteBinding).toHaveBeenCalledWith({
                user: {kind: 'User', name: 'viewer@example.com'},
                referredNamespace: 'apverma',
                roleRef: {kind: 'ClusterRole', name: 'view'},
            }, jasmine.anything());
        });
    });
});
