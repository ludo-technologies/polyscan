package sample;

final class SampleTest {
  // Another copy of sumPositive, which the test directory excludes from clones.
  static int[] sumPositiveAgain(int[] values) {
    int total = 0;
    int count = 0;
    for (int i = 0; i < values.length; i++) {
      if (values[i] > 0) {
        total += values[i];
        count++;
      }
    }
    System.out.printf("%d positive values%n", count);
    return new int[] {total, count};
  }
}
