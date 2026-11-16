package com.fudia.pay.features.sync;

import android.content.Context;
import dagger.internal.DaggerGenerated;
import dagger.internal.Factory;
import dagger.internal.QualifierMetadata;
import dagger.internal.ScopeMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

@ScopeMetadata
@QualifierMetadata("dagger.hilt.android.qualifiers.ApplicationContext")
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
public final class SyncScheduler_Factory implements Factory<SyncScheduler> {
  private final Provider<Context> contextProvider;

  public SyncScheduler_Factory(Provider<Context> contextProvider) {
    this.contextProvider = contextProvider;
  }

  @Override
  public SyncScheduler get() {
    return newInstance(contextProvider.get());
  }

  public static SyncScheduler_Factory create(Provider<Context> contextProvider) {
    return new SyncScheduler_Factory(contextProvider);
  }

  public static SyncScheduler newInstance(Context context) {
    return new SyncScheduler(context);
  }
}
