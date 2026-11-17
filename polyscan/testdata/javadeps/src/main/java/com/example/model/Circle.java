package com.example.model;

import static com.example.service.Units.scale;

public final class Circle implements Shape {
    private final double radius;

    public Circle(double radius) {
        this.radius = radius;
    }

    @Override
    public double area() {
        return scale(Math.PI * radius * radius);
    }
}
