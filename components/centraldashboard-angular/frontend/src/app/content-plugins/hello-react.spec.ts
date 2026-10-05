import '../../../plugins/hello-react/src/hello-react';

interface HelloReactTestElement extends HTMLElement {
  namespace?: string;
  theme?: {
    color: { background: string; foreground: string; primary: string };
    darkMode: boolean;
    spacing: { unit: string };
  };
}

function assert(condition: unknown, message: string): asserts condition {
  if (!condition) {
    throw new Error(message);
  }
}

describe('dashboard-hello-react', () => {
  afterEach(() => document.body.replaceChildren());

  it('defines the custom element', () => {
    assert(customElements.get('dashboard-hello-react'), 'custom element was not defined');
  });

  it('dispatches ready and re-renders namespace and theme properties', () => {
    const element = document.createElement('dashboard-hello-react') as HelloReactTestElement;
    let ready = false;
    element.addEventListener('ready', () => { ready = true; });
    document.body.appendChild(element);

    element.namespace = 'team-b';
    element.theme = {
      color: { background: '#654321', foreground: '#ffffff', primary: '#fedcba' },
      darkMode: true,
      spacing: { unit: '12px' },
    };

    const panel = element.shadowRoot?.querySelector('.plugin') as HTMLElement;
    assert(ready, 'ready was not dispatched');
    assert(panel.textContent?.includes('team-b'), 'namespace was not rendered');
    assert(panel.getAttribute('style')?.includes('rgb(101, 67, 33)'), 'background was not rendered');
    assert(panel.getAttribute('data-dark-mode') === 'true', 'dark mode was not rendered');
  });
});
