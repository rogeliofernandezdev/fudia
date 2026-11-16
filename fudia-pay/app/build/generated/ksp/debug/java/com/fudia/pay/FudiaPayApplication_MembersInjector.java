package com.fudia.pay;

import androidx.hilt.work.HiltWorkerFactory;
import dagger.MembersInjector;
import dagger.internal.DaggerGenerated;
import dagger.internal.InjectedFieldSignature;
import dagger.internal.QualifierMetadata;
import javax.annotation.processing.Generated;
import javax.inject.Provider;

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
public final class FudiaPayApplication_MembersInjector implements MembersInjector<FudiaPayApplication> {
  private final Provider<HiltWorkerFactory> workerFactoryProvider;

  public FudiaPayApplication_MembersInjector(Provider<HiltWorkerFactory> workerFactoryProvider) {
    this.workerFactoryProvider = workerFactoryProvider;
  }

  public static MembersInjector<FudiaPayApplication> create(
      Provider<HiltWorkerFactory> workerFactoryProvider) {
    return new FudiaPayApplication_MembersInjector(workerFactoryProvider);
  }

  @Override
  public void injectMembers(FudiaPayApplication instance) {
    injectWorkerFactory(instance, workerFactoryProvider.get());
  }

  @InjectedFieldSignature("com.fudia.pay.FudiaPayApplication.workerFactory")
  public static void injectWorkerFactory(FudiaPayApplication instance,
      HiltWorkerFactory workerFactory) {
    instance.workerFactory = workerFactory;
  }
}
