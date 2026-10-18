package dev.derekpoe.smallwinner;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import java.util.TreeMap;

public final class CatalogPolicy {
    public static final int MAX_ENTRIES = 512;
    private static final Comparator<CatalogEntry> ORDER = Comparator
            .comparing((CatalogEntry entry) -> entry.label.toLowerCase(Locale.ROOT))
            .thenComparing(CatalogEntry::key).thenComparing(entry -> entry.label);
    private CatalogPolicy() { }

    public static List<CatalogEntry> normalize(Collection<CatalogEntry> entries, String ownPackage) {
        TreeMap<String, CatalogEntry> components = new TreeMap<>();
        for (CatalogEntry entry : entries) {
            if (entry.packageName.equals(ownPackage)) continue;
            components.merge(entry.key(), entry,
                    (left, right) -> ORDER.compare(left, right) <= 0 ? left : right);
        }
        ArrayList<CatalogEntry> result = new ArrayList<>(components.values());
        result.sort(ORDER);
        return List.copyOf(result.subList(0, Math.min(result.size(), MAX_ENTRIES)));
    }

    /** A cached choice must still appear in a fresh launcher query before it can launch. */
    public static boolean isCurrent(CatalogEntry choice, Collection<CatalogEntry> current) {
        return current.stream().anyMatch(entry -> entry.key().equals(choice.key()));
    }
}
