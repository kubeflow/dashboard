/* eslint-disable max-len */
import '@polymer/test-fixture/test-fixture';
import 'jasmine-ajax';
import {mockIronAjax, yieldForRequests} from '../ajax_test_helper';
import {flush} from '@polymer/polymer/lib/utils/flush.js';

import './dashboard-view';

const FIXTURE_ID = 'manage-users-view-contributor-fixture';
const MU_VIEW_SELECTOR_ID = 'test-manage-users-contributor-view';
const TEMPLATE = `
<test-fixture id="${FIXTURE_ID}">
  <template>
    <manage-users-view-contributor id="${MU_VIEW_SELECTOR_ID}"></manage-users-view-contributor>
  </template>
</test-fixture>
`;
const user = 'test@kubeflow.org';
const ownedNs = {namespace: 'ns1', role: 'owner'};
const contribList = [
    {subject: 'foo@kubeflow.org', role: 'contributor', kind: 'User'},
    {subject: 'bar@kubeflow.org', role: 'viewer', kind: 'User'},
];

describe('Manage Users View Contributor', () => {
    let manageUsersViewContributor;

    beforeAll(() => {
        jasmine.Ajax.install();
        const div = document.createElement('div');
        div.innerHTML = TEMPLATE;
        document.body.appendChild(div);
    });

    beforeEach(() => {
        document.getElementById(FIXTURE_ID).create();
        manageUsersViewContributor = document.getElementById(MU_VIEW_SELECTOR_ID);
    });

    afterEach(() => {
        document.getElementById(FIXTURE_ID).restore();
    });

    afterAll(() => {
        jasmine.Ajax.uninstall();
    });

    it('Should handle errors correctly', async () => {
        mockIronAjax(
            manageUsersViewContributor.$.GetContribsAjax,
            'Failed for test',
            true,
        );

        manageUsersViewContributor.user = user;
        manageUsersViewContributor.ownedNamespace = ownedNs;

        flush();
        await yieldForRequests();

        expect(manageUsersViewContributor.$.ContribError.opened)
            .toBe(
                true,
                'Error toast is not opened',
            );
        expect(manageUsersViewContributor.contribError)
            .toBe('Failed for test');
    });

    ['handleContribCreate', 'handleContribDelete'].forEach((handler) => {
        it(`Should show friendly message on body-less 403 from ${handler}`, () => {
            const fakeEvent = {
                detail: {
                    error: 'The request failed with status code: 403',
                    request: {status: 403, response: null},
                },
            };
            manageUsersViewContributor[handler](fakeEvent);

            expect(manageUsersViewContributor.contribCreateError)
                .toBe('You are not authorized to perform this action.');
        });

        it(`Should preserve backend error on 403 from ${handler}`, () => {
            const backendError = 'RoleBinding operation failed: already exists';
            const fakeEvent = {
                detail: {
                    error: 'The request failed with status code: 403',
                    request: {status: 403, response: {error: backendError}},
                },
            };
            manageUsersViewContributor[handler](fakeEvent);

            expect(manageUsersViewContributor.contribCreateError)
                .toBe(backendError);
        });
    });

    it('Should add contributors correctly', async () => {
        const updatedList = [{subject: 'ap@kubeflow.org', role: 'contributor', kind: 'User'}];
        mockIronAjax(
            manageUsersViewContributor.$.GetContribsAjax,
            contribList,
        );
        mockIronAjax(
            manageUsersViewContributor.$.AddContribAjax,
            updatedList,
        );

        manageUsersViewContributor.user = user;
        manageUsersViewContributor.ownedNamespace = ownedNs;

        flush();
        await yieldForRequests();

        const input = manageUsersViewContributor.shadowRoot.querySelector('md2-input');
        input.value = 'new@google.com';
        input.fireEnter();

        await yieldForRequests();

        expect(manageUsersViewContributor.userContributorList)
            .toEqual(
                updatedList,
                'Invalid list of contributors',
            );
    });

    it('Should remove contributors correctly', async () => {
        const updatedList = [{subject: 'ap@kubeflow.org', role: 'contributor', kind: 'User'}];
        mockIronAjax(
            manageUsersViewContributor.$.GetContribsAjax,
            contribList,
        );
        mockIronAjax(
            manageUsersViewContributor.$.RemoveContribAjax,
            updatedList,
        );

        manageUsersViewContributor.user = user;
        manageUsersViewContributor.ownedNamespace = ownedNs;

        flush();
        await yieldForRequests();

        manageUsersViewContributor.removeContributor(
            {model: {item: contribList[0]}},
        );

        await yieldForRequests();

        expect(manageUsersViewContributor.userContributorList)
            .toEqual(
                updatedList,
                'Invalid list of contributors',
            );
    });

    it('UI State should show contribs when namespace available', async () => {
        mockIronAjax(
            manageUsersViewContributor.$.GetContribsAjax,
            contribList,
        );

        manageUsersViewContributor.user = user;
        manageUsersViewContributor.ownedNamespace = ownedNs;

        flush();
        await yieldForRequests();

        expect(manageUsersViewContributor.shadowRoot.querySelector('h2 > .text').innerText)
            .toBe('Contributors for - ns1');

        expect(manageUsersViewContributor.userContributorList)
            .toEqual(
                contribList,
                'Invalid list of user contributors',
            );
        expect(manageUsersViewContributor.groupContributorList)
            .toEqual([], 'Group contributor list should be empty');
    });

    it('Should add group contributors correctly', async () => {
        const contribList = [{subject: 'ml-team', role: 'contributor', kind: 'Group'}];
        const verificationContribs = [
            {subject: 'ml-team', role: 'contributor', kind: 'Group'},
            {subject: 'data-team', role: 'contributor', kind: 'Group'},
        ];
        mockIronAjax(
            manageUsersViewContributor.$.GetContribsAjax,
            contribList,
        );
        mockIronAjax(
            manageUsersViewContributor.$.AddContribAjax,
            verificationContribs,
        );

        manageUsersViewContributor.user = user;
        manageUsersViewContributor.ownedNamespace = ownedNs;

        flush();
        await yieldForRequests();

        const inputs = manageUsersViewContributor.shadowRoot.querySelectorAll('md2-input');
        const groupInput = inputs[1];
        groupInput.value = 'data-team';
        groupInput.fireEnter();

        await yieldForRequests();

        expect(manageUsersViewContributor.groupContributorList)
            .toEqual(
                verificationContribs,
                'Invalid list of group contributors',
            );
    });

    it('Should remove group contributors correctly', async () => {
        const contribList = [
            {subject: 'ml-team', role: 'contributor', kind: 'Group'},
            {subject: 'data-team', role: 'contributor', kind: 'Group'},
        ];
        const verificationContribs = [{subject: 'data-team', role: 'contributor', kind: 'Group'}];
        mockIronAjax(
            manageUsersViewContributor.$.GetContribsAjax,
            contribList,
        );
        mockIronAjax(
            manageUsersViewContributor.$.RemoveContribAjax,
            verificationContribs,
        );

        manageUsersViewContributor.user = user;
        manageUsersViewContributor.ownedNamespace = ownedNs;

        flush();
        await yieldForRequests();

        const inputs = manageUsersViewContributor.shadowRoot.querySelectorAll('md2-input');
        const groupInput = inputs[1];
        const chip = groupInput.querySelector('paper-chip:nth-of-type(1)');
        chip.fireRemove({});

        await yieldForRequests();

        expect(manageUsersViewContributor.groupContributorList)
            .toEqual(
                verificationContribs,
                'Invalid list of group contributors',
            );
    });
});
