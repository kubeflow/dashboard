import '@polymer/test-fixture/test-fixture';

import {
    APP_CONNECTED_EVENT,
    NAMESPACE_SELECTED_EVENT,
    PARENT_CONNECTED_EVENT,
} from '../library.js';
import {sleep} from '../ajax_test_helper';
import './iframe-container';

const FIXTURE_ID = 'iframe-container-fixture';
const IFRAME_CONTAINER_SELECTOR_ID = 'test-iframe-container';
const TEMPLATE = `
<test-fixture id="${FIXTURE_ID}">
  <template>
    <iframe-container id="${IFRAME_CONTAINER_SELECTOR_ID}"></iframe-container>
  </template>
</test-fixture>
`;

describe('Iframe Container', () => {
    let iframeContainer;
    let postMessageSpy;

    beforeAll(() => {
        const div = document.createElement('div');
        div.innerHTML = TEMPLATE;
        document.body.appendChild(div);
    });

    beforeEach(async () => {
        document.getElementById(FIXTURE_ID).create();
        iframeContainer = document.getElementById(IFRAME_CONTAINER_SELECTOR_ID);

        // Set iframe src and spy on child iframe component
        iframeContainer.src = 'about:test';
        await new Promise((resolve) => {
            iframeContainer.$.iframe.addEventListener('load', () => {
                postMessageSpy = spyOn(iframeContainer.$.iframe.contentWindow,
                    'postMessage');
                resolve();
            });
        });
    });

    afterEach(() => {
        document.getElementById(FIXTURE_ID).restore();
    });

    it('Should replace iframe location when src changes', async () => {
        const locationSpy = jasmine.createSpyObj('spyLocation', ['replace']);
        spyOnProperty(iframeContainer.$.iframe, 'contentWindow')
            .and.returnValue({location: locationSpy});

        iframeContainer.src = 'http://foo.bar';
        iframeContainer.src = 'http://foo.bar';
        iframeContainer.src = 'http://foo.bar?test=1';
        iframeContainer.src = 'http://other.bar/#/path';

        expect(locationSpy.replace).toHaveBeenCalledTimes(3);
        const calledWith = locationSpy.replace.calls.all()
            .map((a) => a.args[0]);
        expect(calledWith).toEqual([
            'http://foo.bar',
            'http://foo.bar?test=1',
            'http://other.bar/#/path',
        ]);
    });

    it('Should reflect iframe URL changes to page property', async () => {
        const fakeLocation = {
            href: 'http://testsite.com/foo/bar?name=blah',
            origin: 'http://testsite.com',
        };
        spyOnProperty(iframeContainer.$.iframe, 'contentWindow').and
            .returnValue({location: fakeLocation});
        expect(iframeContainer.page).toBe(undefined);
        iframeContainer.$.iframe.contentDocument.firstChild.click();
        expect(iframeContainer.page).toBe('/foo/bar?name=blah');
    });

    it('Should reflect iframe URL changes on hashchange event', async () => {
        // Capture the real Window before the spy replaces the property, since
        // hashchange listeners are attached to the Window on load.
        const realContentWindow = iframeContainer.$.iframe.contentWindow;
        const fakeLocation = {
            href: 'http://testsite.com/foo/bar?name=blah',
            origin: 'http://testsite.com',
        };
        spyOnProperty(iframeContainer.$.iframe, 'contentWindow').and
            .returnValue({location: fakeLocation});
        expect(iframeContainer.page).toBe(undefined);
        fakeLocation.href = 'http://testsite.com/foo/bar?name=blah#new-hash';
        realContentWindow.dispatchEvent(new Event('hashchange'));
        expect(iframeContainer.page).toBe('/foo/bar?name=blah#new-hash');
    });

    it('Should reflect iframe URL changes on popstate event', async () => {
        const realContentWindow = iframeContainer.$.iframe.contentWindow;
        const fakeLocation = {
            href: 'http://testsite.com/foo/bar?name=blah',
            origin: 'http://testsite.com',
        };
        spyOnProperty(iframeContainer.$.iframe, 'contentWindow').and
            .returnValue({location: fakeLocation});
        expect(iframeContainer.page).toBe(undefined);
        fakeLocation.href = 'http://testsite.com/previous/page';
        realContentWindow.dispatchEvent(new Event('popstate'));
        expect(iframeContainer.page).toBe('/previous/page');
    });

    it('Should synchronize page property when an HTTP page loads',
        async () => {
            const fakeContentWindow = {
                location: {
                    href: 'http://testsite.com/foo/bar?name=blah',
                    origin: 'http://testsite.com',
                    protocol: 'http:',
                },
                addEventListener: () => {},
                postMessage: () => {},
            };
            spyOnProperty(iframeContainer.$.iframe, 'contentWindow').and
                .returnValue(fakeContentWindow);
            iframeContainer.$.iframe.dispatchEvent(new Event('load'));
            expect(iframeContainer.page).toBe('/foo/bar?name=blah');
        });

    it('Should not synchronize page property when a non-HTTP(S) document ' +
        'loads', async () => {
        const fakeContentWindow = {
            location: {
                href: 'about:blank',
                origin: 'null',
                protocol: 'about:',
            },
            addEventListener: () => {},
            postMessage: () => {},
        };
        spyOnProperty(iframeContainer.$.iframe, 'contentWindow').and
            .returnValue(fakeContentWindow);
        iframeContainer.$.iframe.dispatchEvent(new Event('load'));
        expect(iframeContainer.page).toBe(undefined);
    });

    it('Should send messages to iframed page', async () => {
        const origin = window.location.origin;
        // Simulate message being sent from iframe
        window.postMessage({type: APP_CONNECTED_EVENT});
        await sleep(1);
        expect(postMessageSpy.calls.count()).toBe(1);
        expect(postMessageSpy.calls.argsFor(0)).toEqual([
            {
                type: PARENT_CONNECTED_EVENT,
                value: null,
            },
            origin,
        ]);

        // Set namespace
        iframeContainer.namespace = 'test-namespace';
        expect(postMessageSpy.calls.count()).toBe(2);
        expect(postMessageSpy.calls.argsFor(1)).toEqual([
            {
                type: NAMESPACE_SELECTED_EVENT,
                value: 'test-namespace',
            },
            origin,
        ]);
    });
});

// Drives the component with real navigations inside a served fixture page
// instead of a mocked contentWindow. The mocked hashchange test above passed
// for years while the listener was attached to the Document, where Window
// events never fire in a real browser; these tests close that blind spot.
describe('Iframe Container real navigation', () => {
    const FIXTURE_URL = '/base/test_fixtures/hash-routed-app.html';
    const FIXTURE_SECOND_PAGE_URL =
        '/base/test_fixtures/hash-routed-app-second-page.html';

    let container;

    const waitForPage = (expectedPage, timeoutMilliseconds = 3000) =>
        new Promise((resolve, reject) => {
            const startTime = Date.now();
            const poll = () => {
                if (container.page === expectedPage) return resolve();
                if (Date.now() - startTime > timeoutMilliseconds) {
                    return reject(new Error(
                        `timed out waiting for page "${expectedPage}", ` +
                        `page is "${container.page}"`));
                }
                setTimeout(poll, 25);
            };
            poll();
        });

    const loadFixture = (url) => new Promise((resolve) => {
        container.$.iframe.addEventListener('load', resolve, {once: true});
        container.src = url;
    });

    beforeEach(() => {
        container = document.createElement('iframe-container');
        document.body.appendChild(container);
    });

    afterEach(() => {
        document.body.removeChild(container);
    });

    it('Should sync page when a hash link is clicked inside the iframe',
        async () => {
            await loadFixture(FIXTURE_URL);
            expect(container.page).toBe(FIXTURE_URL);
            container.$.iframe.contentDocument
                .getElementById('details-link').click();
            await waitForPage(`${FIXTURE_URL}#/details/123`);
        });

    it('Should sync page on history back inside the iframe', async () => {
        await loadFixture(FIXTURE_URL);
        container.$.iframe.contentDocument
            .getElementById('details-link').click();
        await waitForPage(`${FIXTURE_URL}#/details/123`);
        container.$.iframe.contentWindow.history.back();
        await waitForPage(FIXTURE_URL);
        expect(container.page).toBe(FIXTURE_URL);
    });

    it('Should re-attach listeners after a full document navigation inside ' +
        'the iframe', async () => {
        await loadFixture(FIXTURE_URL);
        const nextLoad = new Promise((resolve) => {
            container.$.iframe.addEventListener('load', resolve, {once: true});
        });
        container.$.iframe.contentDocument
            .getElementById('second-page-link').click();
        await nextLoad;
        await waitForPage(FIXTURE_SECOND_PAGE_URL);
        container.$.iframe.contentDocument
            .getElementById('second-details-link').click();
        await waitForPage(`${FIXTURE_SECOND_PAGE_URL}#/second/456`);
        expect(container.page).toBe(`${FIXTURE_SECOND_PAGE_URL}#/second/456`);
    });
});
