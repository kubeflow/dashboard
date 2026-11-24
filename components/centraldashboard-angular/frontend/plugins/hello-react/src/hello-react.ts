import React from 'react';
import { flushSync } from 'react-dom';
import { createRoot, type Root } from 'react-dom/client';

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

interface HelloReactViewProps {
  namespace?: string;
  namespaces?: string[];
  user?: { email: string };
  theme: DashboardTheme;
  onNavigate: () => void;
  onError: () => void;
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

function HelloReactView(props: HelloReactViewProps) {
  const namespace = props.namespace || props.namespaces?.join(', ') || 'No namespace selected';
  return React.createElement(
    'section',
    {
      className: 'plugin',
      'data-dark-mode': String(props.theme.darkMode),
      style: {
        backgroundColor: props.theme.color.background,
        border: `1px solid ${props.theme.color.primary}`,
        color: props.theme.color.foreground,
        colorScheme: props.theme.darkMode ? 'dark' : 'light',
        padding: props.theme.spacing.unit,
      },
    },
    React.createElement('h2', null, 'Hello from React'),
    React.createElement('p', null, `Namespace: ${namespace}`),
    React.createElement('p', null, `User: ${props.user?.email || 'Unknown user'}`),
    React.createElement('button', { onClick: props.onNavigate, type: 'button' }, 'Navigate'),
    React.createElement('button', { onClick: props.onError, type: 'button' }, 'Report error'),
  );
}

export class HelloReactElement extends HTMLElement {
  private readonly root: Root;
  private currentNamespace?: string;
  private currentNamespaces?: string[];
  private currentTheme: DashboardTheme = defaultTheme;
  private currentUser?: { email: string };
  private currentBasePath?: string;
  private hasRendered = false;

  constructor() {
    super();
    this.root = createRoot(this.attachShadow({ mode: 'open' }));
  }

  get namespace(): string | undefined {
    return this.currentNamespace;
  }

  set namespace(value: string | undefined) {
    this.currentNamespace = value;
    this.render();
  }

  get namespaces(): string[] | undefined {
    return this.currentNamespaces;
  }

  set namespaces(value: string[] | undefined) {
    this.currentNamespaces = value;
    this.render();
  }

  get user(): { email: string } | undefined {
    return this.currentUser;
  }

  set user(value: { email: string } | undefined) {
    this.currentUser = value;
    this.render();
  }

  get theme(): DashboardTheme {
    return this.currentTheme;
  }

  set theme(value: DashboardTheme | undefined) {
    this.currentTheme = value || defaultTheme;
    this.render();
  }

  get basePath(): string | undefined {
    return this.currentBasePath;
  }

  set basePath(value: string | undefined) {
    this.currentBasePath = value;
    this.render();
  }

  connectedCallback() {
    this.render();
  }

  private render() {
    if (!this.isConnected) {
      return;
    }
    flushSync(() => {
      this.root.render(React.createElement(HelloReactView, {
        namespace: this.namespace,
        namespaces: this.namespaces,
        onError: () => this.reportError(),
        onNavigate: () => this.navigate(),
        theme: this.theme,
        user: this.user,
      }));
    });
    if (!this.hasRendered) {
      this.hasRendered = true;
      this.dispatchEvent(new CustomEvent('ready'));
    }
  }

  private navigate() {
    this.dispatchEvent(new CustomEvent('navigate', {
      bubbles: true,
      composed: true,
      detail: { path: '/', queryParams: { from: 'hello-react' } },
    }));
  }

  private reportError() {
    this.dispatchEvent(new CustomEvent('error', {
      bubbles: true,
      composed: true,
      detail: { message: 'Hello React reported a demo error.' },
    }));
  }
}

if (!customElements.get('dashboard-hello-react')) {
  customElements.define('dashboard-hello-react', HelloReactElement);
}
