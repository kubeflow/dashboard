import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, convertToParamMap, ParamMap, Router } from '@angular/router';
import { ReplaySubject } from 'rxjs';
import { EnvironmentService } from 'src/app/services/environment.service';
import { CDBNamespaceService } from 'src/app/services/namespace.service';
import { ContentPlugin, DashboardLinks } from 'src/app/types/dashboard-links';
import { DashboardSettings } from 'src/app/types/dashboard-settings';
import { Namespace } from 'src/app/types/namespace';
import { ContentPluginHostComponent } from './content-plugin-host.component';

interface TestPluginElement extends HTMLElement {
  namespace?: string;
  namespaces?: string[];
  user?: { email: string };
  basePath?: string;
}

describe('ContentPluginHostComponent', () => {
  let component: ContentPluginHostComponent;
  let fixture: ComponentFixture<ContentPluginHostComponent>;
  let routeParams: ReplaySubject<ParamMap>;
  let dashboardLinks: ReplaySubject<DashboardLinks>;
  let dashboardSettings: ReplaySubject<DashboardSettings>;
  let user: ReplaySubject<string>;
  let namespaces: ReplaySubject<Namespace[]>;
  let currentNamespace: ReplaySubject<Namespace>;
  let router: { navigate: jasmine.Spy };
  let injectedScripts: HTMLScriptElement[];

  const pluginId = 'test-plugin';
  const elementName = 'test-content-plugin';

  beforeAll(() => {
    definePluginElement(elementName);
  });

  beforeEach(async () => {
    routeParams = new ReplaySubject<ParamMap>(1);
    dashboardLinks = new ReplaySubject<DashboardLinks>(1);
    dashboardSettings = new ReplaySubject<DashboardSettings>(1);
    user = new ReplaySubject<string>(1);
    namespaces = new ReplaySubject<Namespace[]>(1);
    currentNamespace = new ReplaySubject<Namespace>(1);
    router = { navigate: jasmine.createSpy('navigate') };
    injectedScripts = [];

    await TestBed.configureTestingModule({
      declarations: [ContentPluginHostComponent],
      providers: [
        { provide: ActivatedRoute, useValue: { paramMap: routeParams } },
        { provide: Router, useValue: router },
        {
          provide: EnvironmentService,
          useValue: { dashboardLinks, dashboardSettings, user },
        },
        {
          provide: CDBNamespaceService,
          useValue: {
            namespaces,
            currentNamespace,
            ALL_NAMESPACES: 'All namespaces',
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ContentPluginHostComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  afterEach(() => {
    document
      .querySelectorAll('script[data-content-plugin-id^="test-"]')
      .forEach(script => script.remove());
  });

  it('mounts an enabled v1 plugin with the shell context', async () => {
    addLoadedPluginScript(validPlugin());
    emitNamespace('team-a');
    emitPluginInputs(validPlugin());
    await flushMount();

    const element = getPluginElement(elementName);
    expect(element).toBeTruthy();
    expect(element.namespace).toBe('team-a');
    expect(element.namespaces).toBeUndefined();
    expect(element.user).toEqual({ email: 'user@example.com' });
    expect(element.basePath).toBe('/plugins/test-plugin');

    element.dispatchEvent(new CustomEvent('ready'));
    expect(component.loading).toBeFalse();
  });

  it('maps all-namespaces selection to the namespaces property', async () => {
    addLoadedPluginScript(validPlugin());
    emitNamespace('team-a');
    emitPluginInputs(validPlugin());
    await flushMount();
    currentNamespace.next({
      namespace: 'All namespaces',
      role: '',
      user: '',
    });
    await flushMount();

    const element = getPluginElement(elementName);
    expect(element.namespace).toBeUndefined();
    expect(element.namespaces).toEqual(['team-a']);
  });

  it('uses validated plugin navigation requests', async () => {
    addLoadedPluginScript(validPlugin());
    emitNamespace('team-a');
    emitPluginInputs(validPlugin());
    await flushMount();

    const element = getPluginElement(elementName);
    element.dispatchEvent(
      new CustomEvent('navigate', {
        detail: {
          path: '/namespace-needed',
          queryParams: { ns: 'team-a' },
          fragment: 'details',
        },
      }),
    );

    expect(router.navigate).toHaveBeenCalledWith(['/namespace-needed'], {
      queryParams: { ns: 'team-a' },
      fragment: 'details',
    });
  });

  it('rejects registrations outside the v1 route and replacement scope', () => {
    emitPluginInputs({ ...validPlugin(), route: '/home', replaces: 'home' });

    expect(component.error).toBe('This content plugin has an unsupported registration.');
  });

  it('rejects a cross-origin plugin bundle', () => {
    emitPluginInputs({
      ...validPlugin(),
      bundle: 'https://plugins.example.com/test-plugin/main.js',
    });

    expect(component.error).toBe('This content plugin has an unsupported registration.');
  });

  it('rejects a same-origin bundle outside the approved plugin path', () => {
    emitPluginInputs({
      ...validPlugin(),
      bundle: '/unapproved/test-plugin/main.js',
    });

    expect(component.error).toBe('This content plugin has an unsupported registration.');
  });

  it('keeps plugins disabled until the global ConfigMap gate is enabled', () => {
    emitPluginInputs(validPlugin(), false);

    expect(component.error).toBe('This content plugin is not enabled.');
    expect(fixture.nativeElement.querySelector(elementName)).toBeNull();
  });

  it('removes a failed script so the same plugin can retry', async () => {
    const retryPlugin = validPlugin('test-retry-plugin', 'test-retry-element');
    definePluginElement(retryPlugin.element);
    spyOn(document.head, 'appendChild').and.callFake(<T extends Node>(node: T): T => {
      if (node instanceof HTMLScriptElement) {
        injectedScripts.push(node);
      }
      return node;
    });
    emitPluginInputs(retryPlugin);

    const failedScript = getPluginScript(retryPlugin);
    failedScript.dispatchEvent(new Event('error'));
    await flushMount();
    expect(failedScript.dataset.contentPluginId).toBeUndefined();

    dashboardLinks.next(linksFor([retryPlugin]));
    const retryScript = getPluginScript(retryPlugin);
    expect(retryScript).not.toBe(failedScript);
    expect(retryScript.crossOrigin).toBe('anonymous');
    retryScript.dispatchEvent(new Event('load'));
    await flushMount();

    expect(getPluginElement(retryPlugin.element)).toBeTruthy();
  });

  it('mounts when the custom element is defined after bundle load', async () => {
    const delayedPlugin = validPlugin(
      'test-delayed-plugin',
      'test-delayed-element',
    );
    addLoadedPluginScript(delayedPlugin);
    emitPluginInputs(delayedPlugin);

    await Promise.resolve();
    definePluginElement(delayedPlugin.element);
    await nextTask();
    await flushMount();

    expect(getPluginElement(delayedPlugin.element)).toBeTruthy();
  });

  it('fails when the custom element is never defined before timeout', async () => {
    jasmine.clock().install();
    try {
      const undefinedPlugin = validPlugin(
        'test-undefined-plugin',
        'test-undefined-element',
      );
      addLoadedPluginScript(undefinedPlugin);
      emitPluginInputs(undefinedPlugin);
      await Promise.resolve();
      jasmine.clock().tick(10000);
      await flushMount();

      expect(component.error).toBe(
        'This content plugin did not register its custom element in time.',
      );
    } finally {
      jasmine.clock().uninstall();
    }
  });

  it('does not set an error until registration inputs have all resolved', async () => {
    addLoadedPluginScript(validPlugin());
    routeParams.next(convertToParamMap({ pluginId }));
    expect(component.error).toBeUndefined();

    dashboardLinks.next(linksFor([validPlugin()]));
    expect(component.error).toBeUndefined();

    dashboardSettings.next({ DASHBOARD_CONTENT_PLUGINS_ENABLED: true });
    await flushMount();

    expect(component.error).toBeUndefined();
    expect(getPluginElement(elementName)).toBeTruthy();
  });

  it('redirects a namespaced plugin when all namespaces are selected', async () => {
    emitNamespace('All namespaces');
    emitPluginInputs({ ...validPlugin(), namespaced: true });
    await flushMount();

    expect(router.navigate).toHaveBeenCalledWith(['/namespace-needed']);
  });

  it('redirects a namespaced plugin when no namespace is available', async () => {
    namespaces.next([
      { namespace: 'team-a', role: 'owner', user: 'user@example.com' },
    ]);
    emitPluginInputs({ ...validPlugin(), namespaced: true });
    await flushMount();

    expect(router.navigate).toHaveBeenCalledWith(['/namespace-needed']);
  });

  it('allows a non-namespaced plugin without a namespace', async () => {
    addLoadedPluginScript(validPlugin());
    emitPluginInputs(validPlugin());
    await flushMount();

    expect(router.navigate).not.toHaveBeenCalled();
    expect(getPluginElement(elementName)).toBeTruthy();
  });

  function validPlugin(
    id: string = pluginId,
    element: string = elementName,
  ): ContentPlugin {
    return {
      id,
      route: `/plugins/${id}`,
      element,
      bundle: `/assets/plugins/${id}/main.js`,
      contractVersion: 1,
      enabled: true,
    };
  }

  function emitPluginInputs(
    plugin: ContentPlugin,
    enabled: boolean = true,
  ) {
    routeParams.next(convertToParamMap({ pluginId: plugin.id }));
    dashboardLinks.next(linksFor([plugin]));
    dashboardSettings.next({ DASHBOARD_CONTENT_PLUGINS_ENABLED: enabled });
  }

  function emitNamespace(namespace: string) {
    namespaces.next([
      { namespace: 'team-a', role: 'owner', user: 'user@example.com' },
    ]);
    currentNamespace.next({
      namespace,
      role: namespace === 'team-a' ? 'owner' : '',
      user: 'user@example.com',
    });
    user.next('user@example.com');
  }

  function linksFor(contentPlugins: ContentPlugin[]): DashboardLinks {
    return { menuLinks: [], contentPlugins };
  }

  function addLoadedPluginScript(plugin: ContentPlugin) {
    const script = document.createElement('script');
    script.dataset.contentPluginId = plugin.id;
    document.head.appendChild(script);
  }

  function getPluginScript(plugin: ContentPlugin): HTMLScriptElement {
    const script = getPluginScriptOrNull(plugin);
    if (!script) {
      throw new Error(`Expected script for ${plugin.id}.`);
    }
    return script;
  }

  function getPluginScriptOrNull(
    plugin: ContentPlugin,
  ): HTMLScriptElement | null {
    const script = document.querySelector<HTMLScriptElement>(
      `script[data-content-plugin-id="${plugin.id}"]`,
    );
    return script || injectedScripts.find(
      injected => injected.dataset.contentPluginId === plugin.id,
    ) || null;
  }

  function getPluginElement(element: string): TestPluginElement {
    const host: HTMLElement = fixture.nativeElement;
    const plugin = host.querySelector<TestPluginElement>(element);
    if (!plugin) {
      throw new Error(`Expected element ${element}.`);
    }
    return plugin;
  }

  function definePluginElement(element: string) {
    if (!customElements.get(element)) {
      customElements.define(element, class extends HTMLElement {});
    }
  }

  async function flushMount() {
    for (let index = 0; index < 5; index++) {
      await Promise.resolve();
    }
    fixture.detectChanges();
  }

  function nextTask(): Promise<void> {
    return new Promise(resolve => window.setTimeout(resolve));
  }
});
