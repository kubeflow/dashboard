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
    {user: 'foo@kubeflow.org', role: 'contributor'},
    {user: 'bar@kubeflow.org', role: 'viewer'},
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
                'Error toast is not opened'
            );
        expect(manageUsersViewContributor.contribError)
            .toBe('Failed for test');
    });

    it('Should show friendly message on 403 from add', () => {
        // Simulate a 403 iron-ajax error event directly
        const fakeEvent = {
            detail: {
                error: 'The request failed with status code: 403',
                request: {status: 403, response: null},
            },
        };
        manageUsersViewContributor.handleContribCreate(fakeEvent);

        expect(manageUsersViewContributor.contribCreateError)
            .toBe('You are not authorized to perform this action.');
    });

    it('Should add contributors correctly', async () => {
        const updatedList = [{user: 'ap@kubeflow.org', role: 'contributor'}];
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

        expect(manageUsersViewContributor.contributorList)
            .toEqual(
                updatedList,
                'Invalid list of contributors'
            );
    });

    it('Should remove contributors correctly', async () => {
        const updatedList = [{user: 'ap@kubeflow.org', role: 'contributor'}];
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
            {model: {item: contribList[0]}}
        );

        await yieldForRequests();

        expect(manageUsersViewContributor.contributorList)
            .toEqual(
                updatedList,
                'Invalid list of contributors'
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

        expect(manageUsersViewContributor.contributorList)
            .toEqual(
                contribList,
                'Invalid list of contributors'
            );
    });
});
