package dev.derekpoe.smallwinner;

import android.app.Activity;
import android.app.AlertDialog;
import android.os.Bundle;
import android.view.View;
import android.view.ViewGroup;
import android.view.WindowInsets;
import android.webkit.CookieManager;
import android.webkit.WebResourceError;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Button;
import android.widget.LinearLayout;
import android.widget.TextView;
import java.io.ByteArrayInputStream;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/** Native launcher controls are deliberately outside the web content's capabilities. */
public final class MainActivity extends Activity {
    private final ExecutorService catalogWorker = Executors.newSingleThreadExecutor();
    private AppCatalogDb database;
    private AppCatalog catalog;
    private WebView webView;
    private TextView status;
    private Button apps;
    private Button refresh;
    private volatile boolean offline = true;
    private List<CatalogEntry> currentApps = List.of();
    private static final String OFFLINE_HTML = "<!doctype html><html lang='en'><meta charset='utf-8'>"
            + "<meta name='viewport' content='width=device-width, initial-scale=1'>"
            + "<meta http-equiv='Content-Security-Policy' content=\"default-src 'none'\">"
            + "<title>Smallwinner offline</title><h1>Simulator unavailable</h1>"
            + "<p>The fixed loopback simulator is not responding. Use Retry after starting it.</p>"
            + "<p>Apps and Refresh are native controls and remain available offline. "
            + "This release does not manage device services or change device settings.</p></html>";

    @Override public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        database = new AppCatalogDb(getApplicationContext());
        catalog = new AppCatalog(getApplicationContext());
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setOnApplyWindowInsetsListener((view, insets) -> {
            android.graphics.Insets bars = insets.getInsets(WindowInsets.Type.systemBars());
            view.setPadding(bars.left, bars.top, bars.right, bars.bottom);
            return insets;
        });
        status = new TextView(this);
        status.setPadding(16, 8, 16, 8);
        status.setText("Native app catalog; web panel uses the loopback simulator.");
        root.addView(status);
        LinearLayout toolbar = new LinearLayout(this);
        Button retry = button("Retry simulator", toolbar);
        apps = button("Apps", toolbar);
        refresh = button("Refresh apps", toolbar);
        root.addView(toolbar);
        webView = new WebView(this);
        configureWebView();
        root.addView(webView, new LinearLayout.LayoutParams(-1, 0, 1));
        setContentView(root);
        retry.setOnClickListener(view -> loadSimulator());
        refresh.setOnClickListener(view -> refreshCatalog());
        apps.setOnClickListener(view -> showApps());
        apps.setEnabled(false);
        catalogWorker.execute(() -> {
            try {
                List<CatalogEntry> cached = database.loadAll();
                runOnUiThread(() -> publishCatalog(cached, "Cached native catalog"));
            } catch (RuntimeException ignored) { report("Catalog cache could not be read; refresh to retry."); }
        });
        refreshCatalog();
        loadSimulator();
    }

    private Button button(String title, LinearLayout parent) {
        Button button = new Button(this);
        button.setText(title);
        parent.addView(button, new LinearLayout.LayoutParams(0, -2, 1));
        return button;
    }

    private void configureWebView() {
        WebView.setWebContentsDebuggingEnabled(false);
        WebSettings settings = webView.getSettings();
        settings.setJavaScriptEnabled(false);
        settings.setDomStorageEnabled(false);
        settings.setAllowFileAccess(false);
        settings.setAllowContentAccess(false);
        settings.setAllowFileAccessFromFileURLs(false);
        settings.setAllowUniversalAccessFromFileURLs(false);
        settings.setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);
        settings.setCacheMode(WebSettings.LOAD_NO_CACHE);
        settings.setSafeBrowsingEnabled(true);
        settings.setJavaScriptCanOpenWindowsAutomatically(false);
        settings.setSupportMultipleWindows(false);
        webView.setImportantForAutofill(View.IMPORTANT_FOR_AUTOFILL_NO_EXCLUDE_DESCENDANTS);
        CookieManager.getInstance().setAcceptCookie(false);
        CookieManager.getInstance().setAcceptThirdPartyCookies(webView, false);
        webView.setWebViewClient(new WebViewClient() {
            @Override public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
                return offline || !TrustedContent.allowsNavigation(request.getUrl().toString());
            }

            @Override public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
                String url = request.getUrl().toString();
                boolean allowed = !offline && TrustedContent.allowsResource(url)
                        && (!request.isForMainFrame() || TrustedContent.allowsNavigation(url));
                return allowed ? null : blockedResponse();
            }

            @Override public void onReceivedError(WebView view, WebResourceRequest request, WebResourceError error) {
                if (request.isForMainFrame() && TrustedContent.allowsNavigation(request.getUrl().toString())) showOffline();
            }

            @Override public void onReceivedHttpError(WebView view, WebResourceRequest request, WebResourceResponse response) {
                if (request.isForMainFrame() && TrustedContent.allowsNavigation(request.getUrl().toString())) showOffline();
            }
        });
        // Intentionally no JavaScript interface, message bridge, file chooser, or external URL handler.
    }

    private static WebResourceResponse blockedResponse() {
        return new WebResourceResponse("text/plain", "UTF-8", 403, "Blocked",
                Map.of("Cache-Control", "no-store"), new ByteArrayInputStream(new byte[0]));
    }

    private void loadSimulator() {
        webView.stopLoading();
        offline = false;
        webView.getSettings().setJavaScriptEnabled(true);
        webView.loadUrl(TrustedContent.HOME);
    }

    private void showOffline() {
        if (offline || isDestroyed()) return;
        offline = true;
        webView.stopLoading();
        webView.getSettings().setJavaScriptEnabled(false);
        webView.loadDataWithBaseURL(TrustedContent.OFFLINE, OFFLINE_HTML, "text/html", "UTF-8", null);
    }

    private void refreshCatalog() {
        refresh.setEnabled(false);
        catalogWorker.execute(() -> {
            try {
                List<CatalogEntry> found = catalog.queryLaunchableApps();
                database.replaceAll(found);
                runOnUiThread(() -> publishCatalog(found, "Current native catalog"));
            } catch (RuntimeException ignored) { report("App refresh failed; the previous cache is retained."); }
            finally { runOnUiThread(() -> { if (!isDestroyed()) refresh.setEnabled(true); }); }
        });
    }

    private void publishCatalog(List<CatalogEntry> entries, String label) {
        if (isDestroyed()) return;
        currentApps = entries;
        apps.setEnabled(!entries.isEmpty());
        status.setText(label + ": " + entries.size() + " launcher entries (limit " + CatalogPolicy.MAX_ENTRIES + ").");
    }

    private void showApps() {
        List<CatalogEntry> choices = currentApps;
        String[] labels = choices.stream().map(entry -> entry.label + "\n" + entry.packageName).toArray(String[]::new);
        new AlertDialog.Builder(this).setTitle("Native apps on this Android device")
                .setItems(labels, (dialog, index) -> confirmLaunch(choices.get(index)))
                .setNegativeButton("Close", null).show();
    }

    private void confirmLaunch(CatalogEntry choice) {
        new AlertDialog.Builder(this).setTitle("Open " + choice.label + "?")
                .setMessage(choice.packageName + "\n" + choice.activityName)
                .setNegativeButton("Cancel", null)
                .setPositiveButton("Open app", (dialog, which) -> {
                    try { catalog.launch(choice); }
                    catch (RuntimeException ignored) { status.setText("App could not launch. Refresh the native catalog and try again."); }
                }).show();
    }

    private void report(String message) {
        runOnUiThread(() -> { if (!isDestroyed()) status.setText(message); });
    }

    @Override protected void onDestroy() {
        offline = true;
        webView.stopLoading();
        webView.getSettings().setJavaScriptEnabled(false);
        if (webView.getParent() instanceof ViewGroup parent) parent.removeView(webView);
        webView.destroy();
        // Database operations and close run in FIFO order on the same owner thread.
        catalogWorker.execute(database::close);
        catalogWorker.shutdown();
        super.onDestroy();
    }
}
