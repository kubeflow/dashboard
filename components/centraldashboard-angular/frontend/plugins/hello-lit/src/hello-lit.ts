import { css, html, LitElement } from 'lit';

interface DashboardTheme {
  color: {
    background: string;
    foreground: string;
    primary: string;
  };
  darkMode: boolean;
  spacing: {
    unit: string;
  };
}

const defaultTheme: DashboardTheme = {
  color: {
    background: '#ffffff',
    foreground: '#000000',
    primary: '#3f51b5',
  },
  darkMode: false,
  spacing: { unit: '8px' },
};

export class HelloLitElement extends LitElement {
  static styles = css`
    :host { display: block; }
    .plugin { border: 1px solid var(--plugin-primary); padding: var(--plugin-spacing); }
    button { margin-right: var(--plugin-spacing); }
  `;

  private currentNamespace?: string;
  private currentNamespaces?: string[];
  private currentTheme: DashboardTheme = defaultTheme;
  private currentUser?: { email: string };
  private currentBasePath?: string;

  get namespace(): string | undefined {
    return this.currentNamespace;
  }

  set namespace(value: string | undefined) {
    this.currentNamespace = value;
    this.requestUpdate();
  }

  get namespaces(): string[] | undefined {
    return this.currentNamespaces;
  }

  set namespaces(value: string[] | undefined) {
    this.currentNamespaces = value;
    this.requestUpdate();
  }

  get user(): { email: string } | undefined {
    return this.currentUser;
  }

  set user(value: { email: string } | undefined) {
    this.currentUser = value;
    this.requestUpdate();
  }

  get theme(): DashboardTheme {
    return this.currentTheme;
  }

  set theme(value: DashboardTheme | undefined) {
    this.currentTheme = value || defaultTheme;
    this.requestUpdate();
  }

  get basePath(): string | undefined {
    return this.currentBasePath;
  }

  set basePath(value: string | undefined) {
    this.currentBasePath = value;
    this.requestUpdate();
  }

  firstUpdated() {
    this.dispatchEvent(new CustomEvent('ready'));
  }

  render() {
    const namespace = this.namespace || this.namespaces?.join(', ') || 'No namespace selected';
    const theme = this.theme;
    const style = `background:${theme.color.background};color:${theme.color.foreground};color-scheme:${theme.darkMode ? 'dark' : 'light'};--plugin-primary:${theme.color.primary};--plugin-spacing:${theme.spacing.unit}`;

    return html`
      <section class="plugin" data-dark-mode=${String(theme.darkMode)} style=${style}>
        <h2>Hello from Lit</h2>
        <p>Namespace: ${namespace}</p>
        <p>User: ${this.user?.email || 'Unknown user'}</p>
        <button type="button" @click=${this.navigate}>Navigate</button>
        <button type="button" @click=${this.reportError}>Report error</button>
      </section>
    `;
  }

  private navigate() {
    this.dispatchEvent(new CustomEvent('navigate', {
      bubbles: true,
      composed: true,
      detail: { path: '/', queryParams: { from: 'hello-lit' } },
    }));
  }

  private reportError() {
    this.dispatchEvent(new CustomEvent('error', {
      bubbles: true,
      composed: true,
      detail: { message: 'Hello Lit reported a demo error.' },
    }));
  }
}

if (!customElements.get('dashboard-hello-lit')) {
  customElements.define('dashboard-hello-lit', HelloLitElement);
}
