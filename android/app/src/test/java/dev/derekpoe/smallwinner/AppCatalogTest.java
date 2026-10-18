package dev.derekpoe.smallwinner;

import android.app.Application;
import android.content.Intent;
import android.content.pm.ActivityInfo;
import android.content.pm.ApplicationInfo;
import android.content.pm.ResolveInfo;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.robolectric.RobolectricTestRunner;
import org.robolectric.RuntimeEnvironment;
import org.robolectric.annotation.Config;
import static org.robolectric.Shadows.shadowOf;
import static org.junit.Assert.*;

@RunWith(RobolectricTestRunner.class)
@Config(sdk = 32)
public class AppCatalogTest {
    private void register(String pkg, String activity, boolean exported, boolean enabled) {
        ResolveInfo result = new ResolveInfo();
        result.nonLocalizedLabel = "Synthetic app";
        result.activityInfo = new ActivityInfo();
        result.activityInfo.packageName = pkg;
        result.activityInfo.name = activity;
        result.activityInfo.exported = exported;
        result.activityInfo.enabled = enabled;
        result.activityInfo.applicationInfo = new ApplicationInfo();
        result.activityInfo.applicationInfo.packageName = pkg;
        result.activityInfo.applicationInfo.enabled = true;
        shadowOf(RuntimeEnvironment.getApplication().getPackageManager()).addResolveInfoForIntent(
                new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER), result);
    }

    @Test public void queryExcludesPrivateAndDisabledActivities() {
        Application context = RuntimeEnvironment.getApplication();
        register("sample.publicapp", "sample.publicapp.Main", true, true);
        register("sample.privateapp", "sample.privateapp.Main", false, true);
        register("sample.disabledapp", "sample.disabledapp.Main", true, false);
        AppCatalog catalog = new AppCatalog(context);
        assertEquals(1, catalog.queryLaunchableApps().size());
        assertEquals("sample.publicapp", catalog.queryLaunchableApps().get(0).packageName);
    }

    @Test public void confirmedCurrentSelectionUsesExplicitLauncherIntent() {
        Application context = RuntimeEnvironment.getApplication();
        register("sample.publicapp", "sample.publicapp.Main", true, true);
        AppCatalog catalog = new AppCatalog(context);
        catalog.launch(catalog.queryLaunchableApps().get(0));
        Intent launched = shadowOf(context).getNextStartedActivity();
        assertEquals("sample.publicapp", launched.getComponent().getPackageName());
        assertEquals("sample.publicapp.Main", launched.getComponent().getClassName());
        assertEquals(Intent.ACTION_MAIN, launched.getAction());
        assertTrue(launched.hasCategory(Intent.CATEGORY_LAUNCHER));
        assertTrue((launched.getFlags() & Intent.FLAG_ACTIVITY_NEW_TASK) != 0);
        assertNull(launched.getData());
        assertNull(launched.getExtras());
    }

    @Test public void absentSelectionNeverStartsAnActivity() {
        Application context = RuntimeEnvironment.getApplication();
        AppCatalog catalog = new AppCatalog(context);
        assertThrows(IllegalStateException.class,
                () -> catalog.launch(new CatalogEntry("absent.app", "Main", "Missing")));
        assertNull(shadowOf(context).getNextStartedActivity());
    }
}
