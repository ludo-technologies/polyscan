package com.example.service;

import com.example.model.Circle;
import com.example.model.Shape;
import java.util.List;

public class Service {
    public List<Shape> shapes() {
        return List.of(new Circle(1), new Circle(2));
    }
}
