package com.fudia.pay.features.payments

import androidx.room.Dao
import androidx.room.Database
import androidx.room.Entity
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.PrimaryKey
import androidx.room.Query
import androidx.room.RoomDatabase
import kotlinx.coroutines.flow.Flow

/** Cola local persistente: la notificación sobrevive al reinicio del
 * dispositivo y el worker la sincroniza con reintentos y backoff. La clave de
 * idempotencia (providerEventKey) es única en la base y en el backend. */
@Entity(tableName = "pending_events")
data class PendingEvent(
    @PrimaryKey val providerEventKey: String,
    val provider: String,
    val senderName: String,
    val amount: Double,
    val occurredAtMillis: Long,
    val attempts: Int = 0,
    val createdAtMillis: Long,
)

@Dao
interface PendingEventDao {
    @Query("SELECT * FROM pending_events ORDER BY createdAtMillis ASC")
    fun observeAll(): Flow<List<PendingEvent>>

    @Query("SELECT * FROM pending_events ORDER BY createdAtMillis ASC")
    suspend fun all(): List<PendingEvent>

    @Insert(onConflict = OnConflictStrategy.IGNORE)
    suspend fun insert(event: PendingEvent): Long

    @Query("UPDATE pending_events SET attempts = attempts + 1 WHERE providerEventKey = :key")
    suspend fun markAttempt(key: String)

    @Query("DELETE FROM pending_events WHERE providerEventKey = :key")
    suspend fun delete(key: String)

    @Query("SELECT COUNT(*) FROM pending_events")
    fun countFlow(): Flow<Int>
}

@Database(entities = [PendingEvent::class], version = 1, exportSchema = false)
abstract class PayDatabase : RoomDatabase() {
    abstract fun pendingEventDao(): PendingEventDao
}
