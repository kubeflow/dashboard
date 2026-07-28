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
            newContribRole: {type: String, value: 'contributor'},
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
     * Triggers an API call to create a new Contributor
     */
    addNewContrib() {
        const api = this.$.AddContribAjax;
        api.body = {contributor: this.newContribEmail};
        api.generateRequest();
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
     * Triggers an API call to remove a Contributor
     * @param {Event} e
     */
    removeContributor(e) {
        this._removingRole = e.model.item.role;
        const api = this.$.RemoveContribAjax;
        api.body = {contributor: e.model.item.user};
        api.generateRequest();
    }
    /**
     * Takes an event from iron-ajax and isolates the error from a request that
     * failed
     * @param {IronAjaxEvent} e
     * @return {string}
     */
    _isolateErrorFromIronRequest(e) {
        const status = e.detail.request.status;
        if (status === 403) {
            return 'You are not authorized to perform this action.';
        }
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
        this.contributorList = e.detail.response;
        this.newContribEmail = this.contribCreateError = '';
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
        this.contributorList = e.detail.response;
        this.newContribEmail = this.contribCreateError = '';
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
