package dev.derekpoe.smallwinner;

import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.content.pm.ActivityInfo;
import android.content.pm.PackageManager;
import android.content.pm.ResolveInfo;
import android.os.Build;
import java.util.ArrayList;
import java.util.List;

/** PackageManager query and explicit launch retained from the original native catalog. */
public final class AppCatalog {
    private final Context context;
    public AppCatalog(Context context) { this.context = context.getApplicationContext(); }

    public List<CatalogEntry> queryLaunchableApps() {
        PackageManager manager = context.getPackageManager();
        Intent query = new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER);
        List<ResolveInfo> matches;
        if (Build.VERSION.SDK_INT >= 33) {
            matches = manager.queryIntentActivities(query, PackageManager.ResolveInfoFlags.of(0));
        } else {
            matches = manager.queryIntentActivities(query, 0);
        }
        ArrayList<CatalogEntry> entries = new ArrayList<>();
        for (ResolveInfo match : matches) {
            ActivityInfo info = match.activityInfo;
            if (info == null || !info.enabled || !info.exported
                    || info.applicationInfo == null || !info.applicationInfo.enabled) continue;
            CharSequence rawLabel = match.loadLabel(manager);
            String label = rawLabel == null ? info.packageName : rawLabel.toString();
            if (label.length() > 120) label = label.substring(0, 120);
            try { entries.add(new CatalogEntry(info.packageName, info.name, label)); }
            catch (IllegalArgumentException ignored) { /* Invalid external app metadata is excluded. */ }
        }
        return CatalogPolicy.normalize(entries, context.getPackageName());
    }

    public void launch(CatalogEntry choice) {
        if (!CatalogPolicy.isCurrent(choice, queryLaunchableApps())) {
            throw new IllegalStateException("The selected launcher is no longer available");
        }
        Intent launch = new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER)
                .setComponent(new ComponentName(choice.packageName, choice.activityName))
                .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        context.startActivity(launch);
    }
}
