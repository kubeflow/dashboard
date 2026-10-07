import '@polymer/iron-ajax/iron-ajax.js';
import '@polymer/iron-icon/iron-icon.js';
import '@polymer/iron-icons/iron-icons.js';
import '@polymer/iron-icons/social-icons.js';
import '@polymer/paper-toast/paper-toast.js';
import '@polymer/paper-dropdown-menu/paper-dropdown-menu.js';
import '@polymer/paper-listbox/paper-listbox.js';
import '@polymer/paper-item/paper-item.js';

import {html, PolymerElement} from '@polymer/polymer';

import './resources/paper-chip.js';
import './resources/md2-input/md2-input.js';
import css from './manage-users-view-contributor.css';
import template from './manage-users-view-contributor.pug';
import utilitiesMixin from './utilities-mixin.js';
import {templateContent} from './resources/template-utils.js';

export class ManageUsersViewContributor extends utilitiesMixin(PolymerElement) {
    static get template() {
        return html(templateContent(`
            <style>${css.toString()}</style>
            ${template()}
        `));
    }

    /**
     * Object describing property-related metadata used by Polymer features
     */
    static get properties() {
        return {
            user: {type: String, value: 'Loading...'},
            ownedNamespace: {type: Object, value: () => ({})},
            newContribEmail: String,
            newGroupName: String,
            newContribRole: {type: String, value: 'contributor'},
            userContributorList: {type: Array, value: () => []},
            groupContributorList: {type: Array, value: () => []},
            _removingRole: {type: String, value: 'contributor'},
            contribError: Object,
        };
    }

    /**
     * Computes the add URL based on the selected role.
     * @param {string} namespace
     * @param {string} role
     * @return {string}
     */
    _addUrl(namespace, role) {
        const endpoint = role === 'viewer' ? 'add-viewer' : 'add-contributor';
        return `/api/workgroup/${endpoint}/${namespace}`;
    }
    /**
     * Computes the remove URL based on the role being removed.
     * @param {string} namespace
     * @param {string} role
     * @return {string}
     */
    _removeUrl(namespace, role) {
        const endpoint =
            role === 'viewer' ? 'remove-viewer' : 'remove-contributor';
        return `/api/workgroup/${endpoint}/${namespace}`;
    }
    /**
     * Triggers an API call to create a new user Contributor
     */
    addNewContrib() {
        const api = this.$.AddContribAjax;
        api.body = {contributor: this.newContribEmail, cType: 'user'};
        api.generateRequest();
    }
    /**
     * Triggers an API call to create a new group Contributor
     */
    addNewGroupContrib() {
        const api = this.$.AddContribAjax;
        api.body = {contributor: this.newGroupName, cType: 'group'};
        api.generateRequest();
    }
    /**
     * Triggers an API call to remove a user Contributor
     * @param {Event} e
     */
    removeContributor(e) {
        this._removingRole = e.model.item.role;
        const api = this.$.RemoveContribAjax;
        api.body = {contributor: e.model.item.subject, cType: 'user'};
        api.generateRequest();
    }
    /**
     * Triggers an API call to remove a group Contributor
     * @param {Event} e
     */
    removeGroupContributor(e) {
        this._removingRole = e.model.item.role;
        const api = this.$.RemoveContribAjax;
        api.body = {contributor: e.model.item.subject, cType: 'group'};
        api.generateRequest();
    }
    /**
     * Splits a contributors response array into user and group lists.
     * @param {Array} contribs
     */
    _updateContributorLists(contribs) {
        this.groupContributorList = contribs
            .filter((c) => c.kind && c.kind.toLowerCase() === 'group');
        this.userContributorList = contribs
            .filter((c) => c.kind && c.kind.toLowerCase() === 'user');
    }
    /**
     * Takes an event from iron-ajax and isolates the error from a request that
     * failed
     * @param {IronAjaxEvent} e
     * @return {string}
     */
    _isolateErrorFromIronRequest(e) {
        const status = e.detail.request.status;
        const bd = e.detail.request.response || {};
        if (status === 403 && !bd.error) {
            return 'You are not authorized to perform this action.';
        }
        return bd.error || e.detail.error || e.detail;
    }
    /**
     * Iron-Ajax response / error handler for addNewContributor
     * @param {IronAjaxEvent} e
     */
    handleContribCreate(e) {
        if (e.detail.error) {
            const error = this._isolateErrorFromIronRequest(e);
            this.contribCreateError = error;
            return;
        }
        this._updateContributorLists(e.detail.response);
        this.newContribEmail = this.newGroupName = this.contribCreateError = '';
    }
    /**
     * Iron-Ajax response / error handler for removeContributor
     * @param {IronAjaxEvent} e
     */
    handleContribDelete(e) {
        if (e.detail.error) {
            const error = this._isolateErrorFromIronRequest(e);
            this.contribCreateError = error;
            return;
        }
        this._updateContributorLists(e.detail.response);
        this.newContribEmail = this.newGroupName = this.contribCreateError = '';
    }
    /**
     * Iron-Ajax response handler for getContributors
     * @param {IronAjaxEvent} e
     */
    handleContribFetch(e) {
        if (e.detail.error) {
            this.onContribFetchError(e);
            return;
        }
        this._updateContributorLists(e.detail.response);
    }
    /**
     * Iron-Ajax error handler for getContributors
     * @param {IronAjaxEvent} e
     */
    onContribFetchError(e) {
        const error = this._isolateErrorFromIronRequest(e);
        this.contribError = error;
        this.$.ContribError.show();
    }
}
/* eslint-disable max-len */
customElements.define('manage-users-view-contributor', ManageUsersViewContributor);
