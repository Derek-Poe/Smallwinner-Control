package dev.derekpoe.smallwinner;

import android.content.ContentValues;
import android.content.Context;
import android.database.Cursor;
import android.database.sqlite.SQLiteDatabase;
import android.database.sqlite.SQLiteOpenHelper;
import java.util.ArrayList;
import java.util.List;

/** Derived from the original Smallwinner transactional catalog cache. Native-only data. */
public final class AppCatalogDb extends SQLiteOpenHelper {
    public AppCatalogDb(Context context) { super(context, "smallwinner_catalog.db", null, 1); }

    @Override public void onCreate(SQLiteDatabase db) {
        db.execSQL("CREATE TABLE apps (pkg TEXT NOT NULL, activity TEXT NOT NULL, "
                + "label TEXT NOT NULL, updated_at INTEGER NOT NULL, PRIMARY KEY(pkg,activity))");
        db.execSQL("CREATE INDEX idx_apps_label ON apps(label)");
    }

    @Override public void onUpgrade(SQLiteDatabase db, int oldVersion, int newVersion) {
        throw new IllegalStateException("No catalog migration is defined");
    }

    public void replaceAll(List<CatalogEntry> entries) {
        if (entries.size() > CatalogPolicy.MAX_ENTRIES) throw new IllegalArgumentException("Catalog too large");
        SQLiteDatabase db = getWritableDatabase();
        db.beginTransaction();
        try {
            db.delete("apps", null, null);
            long now = System.currentTimeMillis();
            for (CatalogEntry entry : entries) {
                ContentValues values = new ContentValues();
                values.put("pkg", entry.packageName);
                values.put("activity", entry.activityName);
                values.put("label", entry.label);
                values.put("updated_at", now);
                db.insertOrThrow("apps", null, values);
            }
            db.setTransactionSuccessful();
        } finally { db.endTransaction(); }
    }

    public List<CatalogEntry> loadAll() {
        ArrayList<CatalogEntry> entries = new ArrayList<>();
        try (Cursor cursor = getReadableDatabase().rawQuery(
                "SELECT pkg,activity,label FROM apps LIMIT " + CatalogPolicy.MAX_ENTRIES, null)) {
            while (cursor.moveToNext()) {
                entries.add(new CatalogEntry(cursor.getString(0), cursor.getString(1), cursor.getString(2)));
            }
        }
        return CatalogPolicy.normalize(entries, "");
    }
}
