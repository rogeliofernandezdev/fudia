package com.fudia.pay.core;

import com.fudia.pay.features.payments.PayDatabase;
import com.fudia.pay.features.payments.PendingEventDao;
import dagger.internal.DaggerGenerated;
import dagger.internal.Factory;
import dagger.internal.Preconditions;
import dagger.internal.QualifierMetadata;
import dagger.internal.ScopeMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

@ScopeMetadata
@QualifierMetadata
@DaggerGenerated
@Generated(
    value = "dagger.internal.codegen.ComponentProcessor",
    comments = "https://dagger.dev"
)
@SuppressWarnings({
    "unchecked",
    "rawtypes",
    "KotlinInternal",
    "KotlinInternalInJava",
    "cast",
    "deprecation"
})
public final class DatabaseModule_ProvidePendingEventDaoFactory implements Factory<PendingEventDao> {
  private final Provider<PayDatabase> databaseProvider;

  public DatabaseModule_ProvidePendingEventDaoFactory(Provider<PayDatabase> databaseProvider) {
    this.databaseProvider = databaseProvider;
  }

  @Override
  public PendingEventDao get() {
    return providePendingEventDao(databaseProvider.get());
  }

  public static DatabaseModule_ProvidePendingEventDaoFactory create(
      Provider<PayDatabase> databaseProvider) {
    return new DatabaseModule_ProvidePendingEventDaoFactory(databaseProvider);
  }

  public static PendingEventDao providePendingEventDao(PayDatabase database) {
    return Preconditions.checkNotNullFromProvides(DatabaseModule.INSTANCE.providePendingEventDao(database));
  }
}
