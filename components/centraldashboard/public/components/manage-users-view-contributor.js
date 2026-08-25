import '@polymer/iron-ajax/iron-ajax.js';
import '@polymer/iron-icon/iron-icon.js';
import '@polymer/iron-icons/iron-icons.js';
import '@polymer/iron-icons/social-icons.js';
import '@polymer/paper-toast/paper-toast.js';
import '@polymer/paper-ripple/paper-ripple.js';
import '@polymer/paper-item/paper-icon-item.js';
import '@polymer/paper-icon-button/paper-icon-button.js';

import {html, PolymerElement} from '@polymer/polymer';

import './resources/paper-chip.js';
import './resources/md2-input/md2-input.js';
import css from './manage-users-view-contributor.css';
import template from './manage-users-view-contributor.pug';
import utilitiesMixin from './utilities-mixin.js';

export class ManageUsersViewContributor extends utilitiesMixin(PolymerElement) {
    static get template() {
        return html([`
            <style>${css.toString()}</style>
            ${template()}
        `]);
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
            userContributorList: {type: Array, value: () => []},
            groupContributorList: {type: Array, value: () => []},
            contribError: Object,
            contributorInputEl: Object,
        };
    }
    /**
     * Main ready method for Polymer Elements.
     */
    ready() {
        super.ready();
        this.contributorInputEl = this.$.ContribEmail;
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
        const api = this.$.RemoveContribAjax;
        api.body = {contributor: e.model.item, cType: 'user'};
        api.generateRequest();
    }
    /**
     * Triggers an API call to remove a group Contributor
     * @param {Event} e
     */
    removeGroupContributor(e) {
        const api = this.$.RemoveContribAjax;
        api.body = {contributor: e.model.item, cType: 'group'};
        api.generateRequest();
    }
    /**
     * Splits a contributors response array into user and group lists.
     * @param {Array} contribs
     */
    _updateContributorLists(contribs) {
        this.groupContributorList = contribs
            .filter((c) => c.kind && c.kind.toLowerCase() === 'group')
            .map((c) => c.name);
        this.userContributorList = contribs
            .filter((c) => c.kind && c.kind.toLowerCase() === 'user')
            .map((c) => c.name);
    }
    /**
     * Takes an event from iron-ajax and isolates the error from a request that
     * failed
     * @param {IronAjaxEvent} e
     * @return {string}
     */
    _isolateErrorFromIronRequest(e) {
        const bd = e.detail.request.response||{};
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
