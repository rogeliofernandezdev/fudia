package com.fudia.pay.features.sync;

import com.fudia.pay.core.network.FudiaPayApi;
import com.fudia.pay.features.payments.PendingEventDao;
import dagger.internal.DaggerGenerated;
import dagger.internal.Factory;
import dagger.internal.QualifierMetadata;
import dagger.internal.ScopeMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

@ScopeMetadata("javax.inject.Singleton")
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
public final class PaymentSyncer_Factory implements Factory<PaymentSyncer> {
  private final Provider<PendingEventDao> daoProvider;

  private final Provider<FudiaPayApi> apiProvider;

  public PaymentSyncer_Factory(Provider<PendingEventDao> daoProvider,
      Provider<FudiaPayApi> apiProvider) {
    this.daoProvider = daoProvider;
    this.apiProvider = apiProvider;
  }

  @Override
  public PaymentSyncer get() {
    return newInstance(daoProvider.get(), apiProvider.get());
  }

  public static PaymentSyncer_Factory create(Provider<PendingEventDao> daoProvider,
      Provider<FudiaPayApi> apiProvider) {
    return new PaymentSyncer_Factory(daoProvider, apiProvider);
  }

  public static PaymentSyncer newInstance(PendingEventDao dao, FudiaPayApi api) {
    return new PaymentSyncer(dao, api);
  }
}
