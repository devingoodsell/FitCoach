package pro.d11l.fitcoach.core.db

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase

@Database(entities = [MemorySectionEntity::class], version = 1, exportSchema = false)
abstract class FitCoachDatabase : RoomDatabase() {
    abstract fun memorySectionDao(): MemorySectionDao

    companion object {
        fun create(context: Context): FitCoachDatabase =
            Room.databaseBuilder(context, FitCoachDatabase::class.java, "fitcoach.db").build()
    }
}
