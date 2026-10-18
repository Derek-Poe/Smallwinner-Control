package dev.derekpoe.smallwinner;

/** A launcher component, never a shell command or caller-supplied intent. */
public final class CatalogEntry {
    public final String packageName;
    public final String activityName;
    public final String label;

    public CatalogEntry(String packageName, String activityName, String label) {
        this.packageName = checked(packageName);
        this.activityName = checked(activityName);
        String display = label == null || label.trim().isEmpty() ? packageName : label;
        this.label = display.replaceAll("[\\p{Cntrl}\\p{Cf}]", " ").trim();
        if (this.label.length() > 120) throw new IllegalArgumentException("Label exceeds limit");
    }

    private static String checked(String value) {
        if (value == null || value.isEmpty() || value.length() > 255
                || !value.matches("[A-Za-z0-9_.$]+")) {
            throw new IllegalArgumentException("Invalid launcher component");
        }
        return value;
    }

    public String key() { return packageName + "/" + activityName; }
}
