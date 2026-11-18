package com.fudia.pay.core

import android.content.Context
import androidx.room.Room
import com.fudia.pay.features.payments.PayDatabase
import com.fudia.pay.features.payments.PAY_MIGRATION_1_2
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {
    @Provides
    @Singleton
    fun provideDatabase(@ApplicationContext context: Context): PayDatabase =
        Room.databaseBuilder(context, PayDatabase::class.java, "fudia-pay.db")
            .addMigrations(PAY_MIGRATION_1_2).build()

    @Provides
    fun providePendingEventDao(database: PayDatabase) = database.pendingEventDao()
}
