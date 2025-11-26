import express from 'express';
import {get} from 'http';
import {Request, Response, NextFunction} from 'express';

import {Api} from './api';
import {DefaultApi} from './clients/profile_controller';
import {KubernetesService} from './k8s_service';
import {Interval, MetricsService} from './metrics_service';
import {WorkgroupApi, WorkgroupInfo, SimpleBinding} from './api_workgroup';

describe('Main API', () => {
  let mockK8sService: jasmine.SpyObj<KubernetesService>;
  let mockMetricsService: jasmine.SpyObj<MetricsService>;
  let mockProfilesService: jasmine.SpyObj<DefaultApi>;
  let testApp: express.Application;
  let port: number;
  const newAPI = (withMetrics = false) => new Api(
    mockK8sService,
    withMetrics ? mockMetricsService : undefined,
  );

  describe('Without a Metrics Service', () => {
    beforeEach(() => {
      mockK8sService = jasmine.createSpyObj<KubernetesService>(['']);
      mockProfilesService = jasmine.createSpyObj<DefaultApi>(['']);

      testApp = express();
      testApp.use(express.json());
      testApp.use(
          '/api', newAPI().routes());
      const addressInfo = testApp.listen(0).address();
      if (typeof addressInfo === 'string') {
        throw new Error(
            'Unable to determine system-assigned port for test API server');
      }
      port = addressInfo.port;
    });

    it('Should return a 405 status code', async () => {
        const metricsEndpoint = new Promise((resolve) => {
            get(`http://localhost:${port}/api/metrics`, (res) => {
                expect(res.statusCode).toBe(405);
                resolve();
            });
        });

      const metricsTypeEndpoint = new Promise((resolve) => {
          get(`http://localhost:${port}/api/metrics/podcpu`, (res) => {
              expect(res.statusCode).toBe(405);
              resolve();
          });
      });

      await Promise.all([metricsEndpoint, metricsTypeEndpoint]);
    });
  });

  describe('With a Metrics Service', () => {
    beforeEach(() => {
      mockK8sService = jasmine.createSpyObj<KubernetesService>(['']);
      mockProfilesService = jasmine.createSpyObj<DefaultApi>(['']);
      mockMetricsService = jasmine.createSpyObj<MetricsService>([
        'getNodeCpuUtilization', 'getPodCpuUtilization', 'getPodMemoryUsage', 'getChartsLink'
      ]);

      testApp = express();
      testApp.use(express.json());
      testApp.use(
          '/api',
          newAPI(true).routes());
      const addressInfo = testApp.listen(0).address();
      if (typeof addressInfo === 'string') {
        throw new Error(
            'Unable to determine system-assigned port for test API server');
      } else {
        port = addressInfo.port;
      }
    });

    it('Should retrieve charts link in Metrics service', (done) => {
        get(`http://localhost:${port}/api/metrics`, (res) => {
            expect(res.statusCode).toBe(200);
            expect(mockMetricsService.getChartsLink)
                .toHaveBeenCalled();
            done();
        });
    });

    it('Should retrieve Node CPU Utilization for default 15m interval',
       async () => {
         const defaultInterval = new Promise((resolve) => {
           get(`http://localhost:${port}/api/metrics/node`, (res) => {
             expect(res.statusCode).toBe(200);
             expect(mockMetricsService.getNodeCpuUtilization)
                 .toHaveBeenCalledWith(Interval.Last15m);
             resolve();
           });
         });
         const invalidQsInterval = new Promise((resolve) => {
           get(`http://localhost:${port}/api/metrics/node?interval=100`,
               (res) => {
                 expect(res.statusCode).toBe(200);
                 expect(mockMetricsService.getNodeCpuUtilization)
                     .toHaveBeenCalledWith(Interval.Last15m);
                 resolve();
               });
         });
         await Promise.all([defaultInterval, invalidQsInterval]);
       });

    it('Should retrieve Pod CPU Utilization for default 15m interval',
       (done) => {
         get(`http://localhost:${port}/api/metrics/podcpu`, (res) => {
           expect(res.statusCode).toBe(200);
           expect(mockMetricsService.getPodCpuUtilization)
               .toHaveBeenCalledWith(Interval.Last15m);
           done();
         });
       });

    it('Should retrieve Pod Memory Usage for default 15m interval', (done) => {
      get(`http://localhost:${port}/api/metrics/podmem`, (res) => {
        expect(res.statusCode).toBe(200);
        expect(mockMetricsService.getPodMemoryUsage)
            .toHaveBeenCalledWith(Interval.Last15m);
        done();
      });
    });

    it('Should retrieve Node CPU Utilization for a user-specified interval',
       (done) => {
         get(`http://localhost:${port}/api/metrics/node?interval=Last60m`,
             (res) => {
               expect(res.statusCode).toBe(200);
               expect(mockMetricsService.getNodeCpuUtilization)
                   .toHaveBeenCalledWith(Interval.Last60m);
               done();
             });
       });
  });

  describe('checkNamespaceAccess middleware', () => {
    let mockWorkgroupApi: jasmine.SpyObj<WorkgroupApi>;
    let api: Api;
    let mockReq: Partial<Request>;
    let mockRes: Partial<Response>;
    let mockNext: jasmine.Spy<NextFunction>;
    let jsonSpy: jasmine.Spy;
    let statusSpy: jasmine.Spy;

    beforeEach(() => {
      mockK8sService = jasmine.createSpyObj<KubernetesService>(['']);
      mockWorkgroupApi = jasmine.createSpyObj<WorkgroupApi>(['getWorkgroupInfo']);

      jsonSpy = jasmine.createSpy('json');
      statusSpy = jasmine.createSpy('status').and.returnValue({json: jsonSpy});

      mockRes = {
        status: statusSpy,
        json: jsonSpy,
      };

      mockNext = jasmine.createSpy('next');

      api = new Api(mockK8sService, undefined, mockWorkgroupApi);
    });

    it('should return 400 if namespace parameter is missing', async () => {
      mockReq = {
        params: {},
      };

      // Access the private method via reflection for testing
      await (api as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(statusSpy).toHaveBeenCalledWith(400);
      expect(jsonSpy).toHaveBeenCalledWith({
        error: 'Namespace parameter is required',
      });
      expect(mockNext).not.toHaveBeenCalled();
    });

    it('should allow access if no workgroup API is configured', async () => {
      const apiWithoutWorkgroup = new Api(mockK8sService, undefined, undefined);
      mockReq = {
        params: {namespace: 'test-namespace'},
      };

      await (apiWithoutWorkgroup as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(statusSpy).not.toHaveBeenCalled();
    });

    it('should return 401 if no user is attached to request', async () => {
      mockReq = {
        params: {namespace: 'test-namespace'},
        user: undefined,
      };

      await (api as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(statusSpy).toHaveBeenCalledWith(401);
      expect(jsonSpy).toHaveBeenCalledWith({
        error: 'Authentication required to access namespace activities',
      });
      expect(mockNext).not.toHaveBeenCalled();
    });

    it('should allow access for non-authenticated users in basic auth mode', async () => {
      mockReq = {
        params: {namespace: 'test-namespace'},
        user: {hasAuth: false},
      };

      await (api as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(mockNext).toHaveBeenCalled();
      expect(statusSpy).not.toHaveBeenCalled();
    });

    it('should allow access for cluster admins', async () => {
      const workgroupInfo: WorkgroupInfo = {
        isClusterAdmin: true,
        namespaces: [],
      };

      mockWorkgroupApi.getWorkgroupInfo.and.returnValue(Promise.resolve(workgroupInfo));

      mockReq = {
        params: {namespace: 'test-namespace'},
        user: {hasAuth: true, email: 'admin@example.com'},
      };

      await (api as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(mockWorkgroupApi.getWorkgroupInfo).toHaveBeenCalledWith(mockReq.user);
      expect(mockNext).toHaveBeenCalled();
      expect(statusSpy).not.toHaveBeenCalled();
    });

    it('should allow access for users with any binding to the namespace', async () => {
      const namespaces: SimpleBinding[] = [
        {namespace: 'test-namespace', role: 'viewer', user: 'user@example.com'},
      ];
      const workgroupInfo: WorkgroupInfo = {
        isClusterAdmin: false,
        namespaces,
      };

      mockWorkgroupApi.getWorkgroupInfo.and.returnValue(Promise.resolve(workgroupInfo));

      mockReq = {
        params: {namespace: 'test-namespace'},
        user: {hasAuth: true, email: 'user@example.com'},
      };

      await (api as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(mockWorkgroupApi.getWorkgroupInfo).toHaveBeenCalledWith(mockReq.user);
      expect(mockNext).toHaveBeenCalled();
      expect(statusSpy).not.toHaveBeenCalled();
    });

    it('should deny access for users without any binding to the namespace', async () => {
      const namespaces: SimpleBinding[] = [
        {namespace: 'other-namespace', role: 'owner', user: 'user@example.com'},
      ];
      const workgroupInfo: WorkgroupInfo = {
        isClusterAdmin: false,
        namespaces,
      };

      mockWorkgroupApi.getWorkgroupInfo.and.returnValue(Promise.resolve(workgroupInfo));

      mockReq = {
        params: {namespace: 'test-namespace'},
        user: {hasAuth: true, email: 'user@example.com'},
      };

      await (api as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(mockWorkgroupApi.getWorkgroupInfo).toHaveBeenCalledWith(mockReq.user);
      expect(statusSpy).toHaveBeenCalledWith(403);
      expect(jsonSpy).toHaveBeenCalledWith({
        error: `Access denied. You do not have permission to view activities for namespace 'test-namespace'.`,
      });
      expect(mockNext).not.toHaveBeenCalled();
    });

    it('should return 500 if getWorkgroupInfo throws an error', async () => {
      const error = new Error('Service unavailable');
      mockWorkgroupApi.getWorkgroupInfo.and.returnValue(Promise.reject(error));

      spyOn(console, 'error');

      mockReq = {
        params: {namespace: 'test-namespace'},
        user: {hasAuth: true, email: 'user@example.com'},
      };

      await (api as any).checkNamespaceAccess(mockReq, mockRes, mockNext);

      expect(mockWorkgroupApi.getWorkgroupInfo).toHaveBeenCalledWith(mockReq.user);
      expect(console.error).toHaveBeenCalledWith('Error checking namespace access:', error);
      expect(statusSpy).toHaveBeenCalledWith(500);
      expect(jsonSpy).toHaveBeenCalledWith({
        error: 'Unable to verify namespace access permissions',
      });
      expect(mockNext).not.toHaveBeenCalled();
    });
  });
});
