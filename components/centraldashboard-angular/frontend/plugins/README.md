# Example Content Plugins

`npm run build:plugins` creates self-contained bundles in `src/assets/plugins/`.
Angular copies that directory to `/assets/`, so the registered bundle URLs are served by the
dashboard origin. Build the dashboard normally with `npm run build` or start it with `npm run serve`.

For a local manifest with content plugins enabled, use:

```sh
kustomize build components/centraldashboard-angular/manifests/kustomize/overlays/dev
```

Manual check:

1. Deploy the development overlay and open `/plugins/hello-lit` and `/plugins/hello-react`.
2. Confirm each view renders without an iframe or console error and its loading indicator clears.
3. Change between one namespace and all namespaces; confirm the displayed namespace updates.
4. Confirm the page colors follow the supplied theme, including the dark-mode marker.
5. Select **Navigate** and confirm the dashboard route changes; select **Report error** and confirm the host error is shown.