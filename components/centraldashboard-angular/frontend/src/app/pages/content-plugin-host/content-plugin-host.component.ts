import {
  Component,
  ElementRef,
  OnDestroy,
  OnInit,
  ViewChild,
} from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { combineLatest, Subscription } from 'rxjs';
import { EnvironmentService } from 'src/app/services/environment.service';
import { CDBNamespaceService } from 'src/app/services/namespace.service';
import { Namespace } from 'src/app/types/namespace';
import { ContentPlugin, DashboardLinks } from 'src/app/types/dashboard-links';
import { DashboardSettings } from 'src/app/types/dashboard-settings';

interface ContentPluginElement extends HTMLElement {
  namespace?: string;
  namespaces?: string[];
  user?: { email: string };
  theme?: ContentPluginTheme;
  basePath?: string;
}

interface ContentPluginTheme {
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

const CONTRACT_VERSION = 1;
const CUSTOM_ELEMENT_TIMEOUT_MS = 10000;

const DEFAULT_THEME: ContentPluginTheme = Object.freeze({
  color: Object.freeze({
    background: '#ffffff',
    foreground: '#000000',
    primary: '#3f51b5',
  }),
  darkMode: false,
  spacing: Object.freeze({ unit: '8px' }),
});

@Component({
  selector: 'app-content-plugin-host',
  templateUrl: './content-plugin-host.component.html',
  styleUrls: ['./content-plugin-host.component.scss'],
})
export class ContentPluginHostComponent implements OnInit, OnDestroy {
  @ViewChild('outlet', { static: true }) outlet: ElementRef<HTMLElement>;

  public error: string | undefined;
  public loading = true;

  private readonly subscriptions = new Subscription();
  private plugins: ContentPlugin[] = [];
  private pluginId: string;
  private pluginInputsReady = false;
  private contentPluginsEnabled = false;
  private namespaceContextReady = false;
  private mountingPluginId?: string;
  private mountedPlugin?: ContentPlugin;
  private element?: ContentPluginElement;
  private currentNamespace?: string;
  private namespaces: Namespace[] = [];
  private user?: string;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private env: EnvironmentService,
    private namespaceService: CDBNamespaceService,
  ) {}

  ngOnInit() {
    this.subscriptions.add(
      this.env.user.subscribe(user => {
        this.user = user;
        this.updateContext();
      }),
    );
    this.subscriptions.add(
      this.namespaceService.namespaces.subscribe((namespaces: Namespace[]) => {
        this.namespaces = namespaces;
        this.namespaceContextReady = true;
        this.updateContext();
        Promise.resolve().then(() => this.mountPlugin());
      }),
    );
    this.subscriptions.add(
      this.namespaceService.currentNamespace.subscribe((namespace: Namespace) => {
        this.currentNamespace = namespace.namespace;
        this.namespaceContextReady = true;
        this.updateContext();
        this.mountPlugin();
      }),
    );
    this.subscriptions.add(
      combineLatest([
        this.route.paramMap,
        this.env.dashboardLinks,
        this.env.dashboardSettings,
      ]).subscribe(([params, links, settings]) => {
        this.pluginId = params.get('pluginId') || '';
        this.plugins = links.contentPlugins || [];
        this.contentPluginsEnabled =
          settings.DASHBOARD_CONTENT_PLUGINS_ENABLED === true;
        this.pluginInputsReady = true;
        this.mountPlugin();
      }),
    );
  }

  ngOnDestroy() {
    this.subscriptions.unsubscribe();
    this.unmountPlugin();
  }

  private mountPlugin() {
    if (!this.pluginInputsReady) {
      return;
    }
    const plugin = this.plugins.find(candidate => candidate.id === this.pluginId);
    if (!this.contentPluginsEnabled || !plugin || !plugin.enabled) {
      this.fail('This content plugin is not enabled.');
      return;
    }
    if (!this.isSupportedPlugin(plugin)) {
      this.fail('This content plugin has an unsupported registration.');
      return;
    }
    if (plugin.namespaced && !this.hasConcreteNamespace()) {
      if (!this.namespaceContextReady) {
        return;
      }
      this.unmountPlugin();
      this.loading = false;
      this.router.navigate(['/namespace-needed']);
      return;
    }
    if (
      this.mountedPlugin?.id === plugin.id ||
      this.mountingPluginId === plugin.id
    ) {
      return;
    }

    this.unmountPlugin();
    this.loading = true;
    this.error = undefined;
    this.mountingPluginId = plugin.id;
    this.loadBundle(plugin)
      .then(() => this.waitForCustomElement(plugin.element))
      .then(elementWasDefined => {
        if (this.pluginId !== plugin.id || !elementWasDefined) {
          if (this.pluginId === plugin.id && !elementWasDefined) {
            this.fail('This content plugin did not register its custom element in time.');
          }
          return;
        }
        this.mountedPlugin = plugin;
        this.element = document.createElement(plugin.element) as ContentPluginElement;
        this.element.addEventListener('ready', this.onReady);
        this.element.addEventListener('navigate', this.onNavigate);
        this.element.addEventListener('error', this.onError);
        this.outlet.nativeElement.appendChild(this.element);
        this.updateContext();
      })
      .catch(() => {
        if (this.pluginId === plugin.id) {
          this.fail('The content plugin bundle could not be loaded.');
        }
      })
      .finally(() => {
        if (this.mountingPluginId === plugin.id) {
          this.mountingPluginId = undefined;
        }
      });
  }

  private unmountPlugin() {
    if (!this.element) {
      return;
    }
    this.element.removeEventListener('ready', this.onReady);
    this.element.removeEventListener('navigate', this.onNavigate);
    this.element.removeEventListener('error', this.onError);
    this.element.remove();
    this.element = undefined;
    this.mountedPlugin = undefined;
  }

  private updateContext() {
    if (!this.element || !this.mountedPlugin) {
      return;
    }
    this.element.user = this.user ? { email: this.user } : undefined;
    this.element.theme = DEFAULT_THEME;
    this.element.basePath = this.mountedPlugin.route;
    if (this.currentNamespace === this.namespaceService.ALL_NAMESPACES) {
      this.element.namespace = undefined;
      this.element.namespaces = this.namespaces
        .map(namespace => namespace.namespace)
        .filter(namespace => namespace !== this.namespaceService.ALL_NAMESPACES);
      return;
    }
    this.element.namespace = this.currentNamespace;
    this.element.namespaces = undefined;
  }

  private loadBundle(plugin: ContentPlugin): Promise<void> {
    const existing = Array.from(document.scripts).find(
      script => script.dataset.contentPluginId === plugin.id,
    );
    if (existing) {
      return Promise.resolve();
    }
    return new Promise((resolve, reject) => {
      const script = document.createElement('script');
      script.src = plugin.bundle;
      script.async = true;
      script.crossOrigin = 'anonymous';
      script.dataset.contentPluginId = plugin.id;
      script.onload = () => resolve();
      script.onerror = () => {
        script.removeAttribute('data-content-plugin-id');
        script.remove();
        reject();
      };
      // Plugins run as trusted first-party code; anonymous mode supports future CSP/SRI hardening.
      document.head.appendChild(script);
    });
  }

  private waitForCustomElement(element: string): Promise<boolean> {
    if (customElements.get(element)) {
      return Promise.resolve(true);
    }
    return new Promise(resolve => {
      const timeout = window.setTimeout(
        () => resolve(false),
        CUSTOM_ELEMENT_TIMEOUT_MS,
      );
      customElements.whenDefined(element).then(() => {
        window.clearTimeout(timeout);
        resolve(true);
      });
    });
  }

  private isSupportedPlugin(plugin: ContentPlugin): boolean {
    if (
      plugin.contractVersion !== CONTRACT_VERSION ||
      plugin.route !== `/plugins/${plugin.id}` ||
      plugin.replaces ||
      !plugin.element.includes('-')
    ) {
      return false;
    }
    try {
      const bundleUrl = new URL(plugin.bundle, window.location.origin);
      const expectedBundlePath = `/assets/plugins/${plugin.id}/main.js`;
      return bundleUrl.origin === window.location.origin &&
        bundleUrl.pathname === expectedBundlePath;
    } catch (_) {
      return false;
    }
  }

  private fail(message: string) {
    this.unmountPlugin();
    this.loading = false;
    this.error = message;
  }

  private onReady = () => {
    this.loading = false;
  };

  private onError = (event: Event) => {
    const detail = (event as CustomEvent<{ message?: string }>).detail;
    this.fail(detail?.message || 'The content plugin reported an error.');
  };

  private onNavigate = (event: Event) => {
    const detail = (event as CustomEvent<{
      path?: string;
      queryParams?: Params;
      fragment?: string;
    }>).detail;
    if (!detail || !this.isSafeNavigation(detail.path)) {
      this.fail('The content plugin requested an invalid route.');
      return;
    }
    this.router.navigate([detail.path], {
      queryParams: detail.queryParams,
      fragment: detail.fragment,
    });
  };

  private isSafeNavigation(path: string | undefined): boolean {
    return !!path && path.startsWith('/') && !path.startsWith('//');
  }

  private hasConcreteNamespace(): boolean {
    return !!this.currentNamespace &&
      this.currentNamespace !== this.namespaceService.ALL_NAMESPACES;
  }
}
