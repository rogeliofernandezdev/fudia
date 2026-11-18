package com.fudia.pay.features.notifications;

import android.content.Context;
import dagger.internal.DaggerGenerated;
import dagger.internal.Factory;
import dagger.internal.QualifierMetadata;
import dagger.internal.ScopeMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

@ScopeMetadata("javax.inject.Singleton")
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
public final class ListenerStatus_Factory implements Factory<ListenerStatus> {
  private final Provider<Context> contextProvider;

  public ListenerStatus_Factory(Provider<Context> contextProvider) {
    this.contextProvider = contextProvider;
  }

  @Override
  public ListenerStatus get() {
    return newInstance(contextProvider.get());
  }

  public static ListenerStatus_Factory create(Provider<Context> contextProvider) {
    return new ListenerStatus_Factory(contextProvider);
  }

  public static ListenerStatus newInstance(Context context) {
    return new ListenerStatus(context);
  }
}
