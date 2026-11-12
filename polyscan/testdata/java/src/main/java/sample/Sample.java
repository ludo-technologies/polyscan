package sample;

final class Sample {
  // Adds the positive values and reports how many there were.
  static int[] sumPositive(int[] values) {
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

  // sumPositive with the names and the comparison changed.
  static int[] sumNegative(int[] numbers) {
    int sum = 0;
    int seen = 0;
    for (int i = 0; i < numbers.length; i++) {
      if (numbers[i] < 0) {
        sum += numbers[i];
        seen++;
      }
    }
    System.out.printf("%d negative values%n", seen);
    return new int[] {sum, seen};
  }

  static final class Server {
    // Two loops, two switch cases, an if, its &&, a ternary and a catch are
    // eight decision points, so the complexity is 9.
    int handle(int[] items, int mode) {
      int acc = 0;
      for (int item : items) {
        acc += item;
      }
      while (acc > 100) {
        acc--;
      }
      switch (mode) {
        case 1:
          acc += 10;
          break;
        case 2:
          acc += 20;
          break;
        default:
          break;
      }
      if (acc > 5 && items.length > 0) {
        acc += items[0];
      }
      try {
        acc += items.length > 0 ? items.length : -items.length;
      } catch (RuntimeException e) {
        acc = 0;
      }
      return acc;
    }
  }
}
