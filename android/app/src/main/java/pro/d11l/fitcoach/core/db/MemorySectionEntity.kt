package pro.d11l.fitcoach.core.db

import androidx.room.Entity
import androidx.room.PrimaryKey

/** Locally cached Coach Memory section for offline reads. */
@Entity(tableName = "memory_sections")
data class MemorySectionEntity(
    @PrimaryKey val section: String,
    val schemaVersion: Int,
    val dataJson: String,
    val updatedAt: String?,
)
