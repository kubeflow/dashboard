import '../../../plugins/hello-lit/src/hello-lit';

interface HelloLitTestElement extends HTMLElement {
  namespace?: string;
  theme?: {
    color: { background: string; foreground: string; primary: string };
    darkMode: boolean;
    spacing: { unit: string };
  };
  updateComplete: Promise<unknown>;
}

function assert(condition: unknown, message: string): asserts condition {
  if (!condition) {
    throw new Error(message);
  }
}

describe('dashboard-hello-lit', () => {
  afterEach(() => document.body.replaceChildren());

  it('defines the custom element', () => {
    assert(customElements.get('dashboard-hello-lit'), 'custom element was not defined');
  });

  it('dispatches ready and re-renders namespace and theme properties', async () => {
    const element = document.createElement('dashboard-hello-lit') as HelloLitTestElement;
    let ready = false;
    element.addEventListener('ready', () => { ready = true; });
    document.body.appendChild(element);
    await element.updateComplete;

    element.namespace = 'team-a';
    element.theme = {
      color: { background: '#123456', foreground: '#ffffff', primary: '#abcdef' },
      darkMode: true,
      spacing: { unit: '12px' },
    };
    await element.updateComplete;

    const panel = element.shadowRoot?.querySelector('.plugin') as HTMLElement;
    assert(ready, 'ready was not dispatched');
    assert(panel.textContent?.includes('team-a'), 'namespace was not rendered');
    assert(panel.getAttribute('style')?.includes('#123456'), 'background was not rendered');
    assert(panel.getAttribute('data-dark-mode') === 'true', 'dark mode was not rendered');
  });
});
