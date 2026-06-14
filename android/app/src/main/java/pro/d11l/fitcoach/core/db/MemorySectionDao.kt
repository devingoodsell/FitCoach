package pro.d11l.fitcoach.core.db

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query

@Dao
interface MemorySectionDao {
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsertAll(sections: List<MemorySectionEntity>)

    @Query("SELECT * FROM memory_sections ORDER BY section")
    suspend fun all(): List<MemorySectionEntity>

    @Query("DELETE FROM memory_sections")
    suspend fun clear()
}
