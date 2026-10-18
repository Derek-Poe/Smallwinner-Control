package dev.derekpoe.smallwinner;

import android.content.Context;
import java.util.List;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.robolectric.RobolectricTestRunner;
import org.robolectric.RuntimeEnvironment;
import org.robolectric.annotation.Config;
import static org.junit.Assert.*;

@RunWith(RobolectricTestRunner.class)
@Config(sdk = 32)
public class AppCatalogDbTest {
    private AppCatalogDb db;
    @Before public void open() {
        Context context = RuntimeEnvironment.getApplication();
        context.deleteDatabase("smallwinner_catalog.db");
        db = new AppCatalogDb(context);
    }
    @After public void close() { db.close(); }

    @Test public void roundTripsDistinctActivitiesAndReplacesCache() {
        db.replaceAll(List.of(new CatalogEntry("sample.app", "A", "Alpha"), new CatalogEntry("sample.app", "B", "Beta")));
        List<CatalogEntry> rows = db.loadAll();
        assertEquals(2, rows.size());
        assertEquals("sample.app/A", rows.get(0).key());
        assertEquals("sample.app/B", rows.get(1).key());
        db.replaceAll(List.of(new CatalogEntry("sample.other", "Main", "Other")));
        assertEquals(1, db.loadAll().size());
        assertEquals("sample.other/Main", db.loadAll().get(0).key());
    }

    @Test public void failedInsertRollsBackDeleteAndEarlierInserts() {
        CatalogEntry old = new CatalogEntry("old.app", "Main", "Old");
        CatalogEntry duplicate = new CatalogEntry("new.app", "Main", "New");
        db.replaceAll(List.of(old));
        assertThrows(android.database.SQLException.class, () -> db.replaceAll(List.of(duplicate, duplicate)));
        assertEquals(1, db.loadAll().size());
        assertEquals(old.key(), db.loadAll().get(0).key());
    }

    @Test public void emptyRefreshIsAnAtomicEmptyCatalog() {
        db.replaceAll(List.of(new CatalogEntry("sample.app", "Main", "App")));
        db.replaceAll(List.of());
        assertTrue(db.loadAll().isEmpty());
    }
}
