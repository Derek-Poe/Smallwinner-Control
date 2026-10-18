package dev.derekpoe.smallwinner;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Locale;
import org.junit.Test;
import static org.junit.Assert.*;

public class CatalogPolicyTest {
    private CatalogEntry row(String pkg, String activity, String label) { return new CatalogEntry(pkg, activity, label); }

    @Test public void keepsDistinctActivitiesDeduplicatesComponentsAndExcludesSelf() {
        List<CatalogEntry> input = List.of(row("sample.a", "A", "One"), row("sample.a", "B", "Two"),
                row("sample.a", "A", "One"), row("self.app", "Main", "Self"));
        List<CatalogEntry> normalized = CatalogPolicy.normalize(input, "self.app");
        assertEquals(2, normalized.size());
        assertEquals("sample.a/A", normalized.get(0).key());
        assertEquals("sample.a/B", normalized.get(1).key());
        assertThrows(UnsupportedOperationException.class, () -> normalized.clear());
    }

    @Test public void selectionRequiresExactCurrentComponent() {
        CatalogEntry choice = row("sample.app", "sample.app.Main", "App");
        assertTrue(CatalogPolicy.isCurrent(choice, List.of(row("sample.app", "sample.app.Main", "Renamed"))));
        assertFalse(CatalogPolicy.isCurrent(choice, List.of(row("sample.app", "sample.app.Other", "App"))));
        assertFalse(CatalogPolicy.isCurrent(choice, List.of(row("other.app", "sample.app.Main", "App"))));
        assertFalse(CatalogPolicy.isCurrent(choice, List.of()));
    }

    @Test public void boundedResultIsStableAcrossEnumerationAndLocale() {
        ArrayList<CatalogEntry> input = new ArrayList<>();
        for (int i = 0; i < 600; i++) input.add(row("sample.p" + i, "Main", "I" + String.format(Locale.ROOT, "%03d", i)));
        Locale before = Locale.getDefault();
        try {
            Locale.setDefault(Locale.forLanguageTag("tr-TR"));
            List<CatalogEntry> first = CatalogPolicy.normalize(input, "self.app");
            Collections.reverse(input);
            Locale.setDefault(Locale.US);
            List<CatalogEntry> second = CatalogPolicy.normalize(input, "self.app");
            assertEquals(512, first.size());
            for (int i = 0; i < first.size(); i++) assertEquals(first.get(i).key(), second.get(i).key());
        } finally { Locale.setDefault(before); }
    }

    @Test public void rejectsCommandLikeComponentsAndBoundsDisplayMetadata() {
        assertThrows(IllegalArgumentException.class, () -> row("sample.app;run", "Main", "App"));
        assertThrows(IllegalArgumentException.class, () -> row("sample.app", "../Main", "App"));
        assertThrows(IllegalArgumentException.class, () -> row("sample.app", "Main", "a".repeat(121)));
        assertEquals("App Name", row("sample.app", "Main", "App\nName").label);
        assertEquals("sample.app", row("sample.app", "Main", " ").label);
    }
}
