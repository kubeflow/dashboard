import {NextFunction, Request} from 'express';

import {attachUser} from './attach_user_middleware';

describe('Attach User Middleware', () => {
  const userIdHeader = 'X-User-Email';
  const prefix = '';
  const groupsHeader = 'kubeflow-groups';
  const middleware = attachUser(userIdHeader, prefix, groupsHeader);

  let mockRequest: jasmine.SpyObj<Request>;
  let mockNextFunction: jasmine.Spy;

  beforeEach(() => {
    mockRequest = jasmine.createSpyObj<Request>('mockRequest', ['header']);
    mockNextFunction = jasmine.createSpy();
    mockRequest.header.withArgs(groupsHeader).and.returnValue(null);
  });

  it('Should extract a User from the request headers when present', () => {
    const email = 'user@domain.com';
    mockRequest.header.withArgs(userIdHeader).and.returnValue(email);

    middleware(mockRequest, null, mockNextFunction);

    expect(mockRequest.user).toEqual({
      auth: {[userIdHeader]: 'user@domain.com'},
      domain: 'domain.com',
      email,
      hasAuth: true,
      username: 'user',
      groups: [],
    });
    expect(mockNextFunction).toHaveBeenCalled();
  });

  it('Should extract a default User when no user header is present in request', () => {
    const email = 'anonymous@kubeflow.org';
    mockRequest.header.withArgs(userIdHeader).and.returnValue(null);

    middleware(mockRequest, null, mockNextFunction);

    expect(mockRequest.user).toEqual({
      auth: undefined,
      domain: 'kubeflow.org',
      email,
      hasAuth: false,
      username: 'anonymous',
      groups: [],
    });
    expect(mockNextFunction).toHaveBeenCalled();
  });

  it('Should extract a default User when no userid header was passed in', () => {
    const email = 'anonymous@kubeflow.org';
    mockRequest.header.withArgs(userIdHeader).and.returnValue(null);
    const noHeaderMiddleware = attachUser('', '', '');
    noHeaderMiddleware(mockRequest, null, mockNextFunction);

    expect(mockRequest.user).toEqual({
      auth: undefined,
      domain: 'kubeflow.org',
      email,
      hasAuth: false,
      username: 'anonymous',
      groups: [],
    });
    expect(mockNextFunction).toHaveBeenCalled();
  });

  it('Should extract groups from the groups header when present', () => {
    const email = 'user@domain.com';
    mockRequest.header.withArgs(userIdHeader).and.returnValue(email);
    mockRequest.header.withArgs(groupsHeader).and.returnValue('group-a, group-b');

    middleware(mockRequest, null, mockNextFunction);

    expect(mockRequest.user.groups).toEqual(['group-a', 'group-b']);
    expect(mockNextFunction).toHaveBeenCalled();
  });

  it('Should extract groups from a JSON array groups header', () => {
    const email = 'user@domain.com';
    mockRequest.header.withArgs(userIdHeader).and.returnValue(email);
    mockRequest.header.withArgs(groupsHeader).and.returnValue('["group-a","group-b"]');

    middleware(mockRequest, null, mockNextFunction);

    expect(mockRequest.user.groups).toEqual(['group-a', 'group-b']);
    expect(mockNextFunction).toHaveBeenCalled();
  });

  it('Should extract groups from a base64-encoded JSON array groups header ' +
      '(as emitted by Istio\'s RequestAuthentication.outputClaimToHeaders ' +
      'for array JWT claims)', () => {
    const email = 'user@domain.com';
    const encoded =
        Buffer.from(JSON.stringify(['group-a', 'group-b'])).toString('base64');
    mockRequest.header.withArgs(userIdHeader).and.returnValue(email);
    mockRequest.header.withArgs(groupsHeader).and.returnValue(encoded);

    middleware(mockRequest, null, mockNextFunction);

    expect(mockRequest.user.groups).toEqual(['group-a', 'group-b']);
    expect(mockNextFunction).toHaveBeenCalled();
  });

  it('Should treat a group name that happens to be valid base64 as a ' +
      'literal group name, not silently decode it to a different identity',
      () => {
    const email = 'user@domain.com';
    // 'dGVhbQ==' is valid base64 for 'team', but it is not JSON, so it must
    // be treated as the literal (single) group name it appears to be.
    mockRequest.header.withArgs(userIdHeader).and.returnValue(email);
    mockRequest.header.withArgs(groupsHeader).and.returnValue('dGVhbQ==');

    middleware(mockRequest, null, mockNextFunction);

    expect(mockRequest.user.groups).toEqual(['dGVhbQ==']);
    expect(mockNextFunction).toHaveBeenCalled();
  });
});
