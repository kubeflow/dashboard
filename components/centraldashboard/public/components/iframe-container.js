/**
 * @fileoverview Component to manage Kubeflow application UIs within an iframe.
 */
import {html, PolymerElement} from '@polymer/polymer';

import {
    APP_CONNECTED_EVENT,
    MESSAGE,
    NAMESPACE_SELECTED_EVENT,
    PARENT_CONNECTED_EVENT,
    ALL_NAMESPACES_EVENT,
} from '../library.js';

import {ALL_NAMESPACES} from './namespace-selector';

export class IframeContainer extends PolymerElement {
    static get template() {
        return html`
            <style>
                :host {
                    flex: 1;
                }

                iframe {
                    border: 0;
                    display: inline-block;
                    height: 100%;
                    width: 100%;
                }
            </style>
            <iframe id="iframe"></iframe>
        `;
    }

    static get properties() {
        return {
            namespaces: Array,
            namespace: {type: String, observer: '_sendNamespaceMessage'},
            src: {type: String, observer: '_srcChanged'},
            page: {type: String, notify: true},
        };
    }

    /**
     * Initializes private iframe state variables and attaches a listener for
     * messages received by the window object.
     */
    ready() {
        super.ready();
        this._iframeOrigin = null;
        this._messageListener = this._onMessageReceived.bind(this);
        window.addEventListener(MESSAGE, this._messageListener);

        // Adds a click handler to be able to capture navigation events from
        // the captured iframe and set the page property which notifies
        const iframe = this.$.iframe;
        this._syncIframePage = () => {
            const iframeLocation = iframe.contentWindow.location;
            const newIframePage = iframeLocation.href.slice(
                iframeLocation.origin.length);
            if (this.page !== newIframePage) {
                this.page = newIframePage;
            }
        };
        iframe.addEventListener('load', () => {
            // Synchronize the page property on load so that server-side
            // navigations are captured before any click or hashchange event.
            // Non-HTTP(S) documents such as about:blank are skipped: their
            // opaque origin serializes to "null", which would produce an
            // invalid page value and corrupt the parent URL through the
            // two-way page binding.
            const {protocol} = iframe.contentWindow.location;
            if (protocol === 'http:' || protocol === 'https:') {
                this._syncIframePage();
            }
            // hashchange and popstate are Window events; a Document listener
            // never receives them, so in-iframe hash navigation (for example
            // the hash-routed Pipelines web application) was never synced.
            // The stable listener reference makes re-attachment on every load
            // a no-op when the browser reuses the Window, and a fresh attach
            // when navigation replaced it.
            const {contentDocument, contentWindow} = iframe;
            contentDocument.addEventListener('click', this._syncIframePage);
            contentWindow.addEventListener('hashchange', this._syncIframePage);
            contentWindow.addEventListener('popstate', this._syncIframePage);
        });
    }

    /**
     * Remove the event listener for messages.
     */
    disconnectedCallback() {
        super.disconnectedCallback();
        window.removeEventListener(MESSAGE, this._messageListener);
    }

    /**
     * Programmatically sets the iframe's src when the property changes.
     * @param {string} newSrc
     */
    _srcChanged(newSrc) {
        const iframe = this.$.iframe;
        if (iframe.contentWindow.location.toString() !== newSrc) {
            iframe.contentWindow.location.replace(newSrc);
        }
    }

    /**
     * Sends a message to the iframe message bus. This is used on the iframe
     * load event as well as when the namespace changes.
     */
    _sendNamespaceMessage() {
        if (!(this._iframeOrigin && this.namespace)) return;
        if (this.namespace === ALL_NAMESPACES) {
            this.$.iframe.contentWindow.postMessage({
                type: ALL_NAMESPACES_EVENT,
                value: this.namespaces.map((n) => n.namespace)
                    .filter((n) => n !== ALL_NAMESPACES),
            }, this._iframeOrigin);
        } else {
            this.$.iframe.contentWindow.postMessage({
                type: NAMESPACE_SELECTED_EVENT,
                value: this.namespace,
            }, this._iframeOrigin);
        }
    }

    /**
     * Receives a message from an iframe page and passes the selected namespace.
     * @param {MessageEvent} event
     */
    _onMessageReceived(event) {
        const {data, origin} = event;
        this._iframeOrigin = origin;
        switch (data.type) {
        case APP_CONNECTED_EVENT:
            this.$.iframe.contentWindow.postMessage({
                type: PARENT_CONNECTED_EVENT,
                value: null,
            }, origin);
            this._sendNamespaceMessage();
            break;
        }
    }
}

window.customElements.define('iframe-container', IframeContainer);
