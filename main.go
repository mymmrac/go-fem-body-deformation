package main

import (
	"log/slog"
	"strconv"

	gui "github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

func main() {
	rl.InitWindow(960, 540, "Body Deformation")
	defer rl.CloseWindow()

	rl.SetTargetFPS(60)
	rl.SetWindowState(rl.FlagWindowResizable)

	camera := rl.NewCamera3D(
		rl.NewVector3(10, 10, 10),
		rl.NewVector3(0, 0, 0),
		rl.NewVector3(0, 1, 0),
		45,
		rl.CameraPerspective,
	)
	cameraOrbiting := false
	// cameraOrbiting := true // TODO: Reset

	// TODO: Reset to [4, 5, 3]
	bodySize := [3]*InputValue[float64]{
		NewInputValue(4.0),
		NewInputValue(4.0),
		NewInputValue(4.0),
	}

	// TODO: Reset to [4, 8, 3]
	bodySplit := [3]*InputValue[int]{
		NewInputValue(2),
		NewInputValue(2),
		NewInputValue(2),
	}

	yungaModule := NewInputValue(4.0)
	poissonRatio := NewInputValue(0.3)
	pressure := NewInputValue(10.0)

	body, bodyOuterIndexes := buildBodyShape(InputsToVec3(bodySize), InputsToVec3(bodySplit))

	const inputTextSize = 20
	var (
		inputWidth  float32 = 64.0
		inputHeight float32 = 24.0
		padding     float32 = 8.0
	)

	a0 := rl.NewVector3(0, 0, 0)
	aX := rl.NewVector3(1, 0, 0)
	aY := rl.NewVector3(0, 1, 0)
	aZ := rl.NewVector3(0, 0, 1)

	showNumbers := false
	var numbers []rl.Texture2D
	for i := range 1 << 11 {
		textImage := rl.ImageTextEx(rl.GetFontDefault(), strconv.Itoa(i), 32, 4, rl.White)
		numbers = append(numbers, rl.LoadTextureFromImage(textImage))
		rl.UnloadImage(textImage)
	}

	for !rl.WindowShouldClose() {
		topLeftUiRect := rl.NewRectangle(
			0, 0,
			padding+inputWidth*3.5+padding*3+padding,
			padding+inputHeight*3+padding*2+padding,
		)
		bottomLeftUiRect := rl.NewRectangle(
			0, float32(rl.GetScreenHeight())-(padding+inputHeight*3+padding*2+padding),
			padding+inputWidth*2+padding+padding,
			padding+inputHeight*3+padding*2+padding,
		)

		if rl.IsKeyPressed(rl.KeySpace) {
			cameraOrbiting = !cameraOrbiting
		}

		if rl.IsMouseButtonDown(rl.MouseButtonLeft) && (!rl.CheckCollisionPointRec(rl.GetMousePosition(), topLeftUiRect) &&
			!rl.CheckCollisionPointRec(rl.GetMousePosition(), bottomLeftUiRect)) {
			md := rl.GetMouseDelta()
			rl.CameraYaw(&camera, -md.X*0.003, 1)
			rl.CameraPitch(&camera, -md.Y*0.003, 1, 1, 0)
		} else if cameraOrbiting {
			var rotation = rl.MatrixRotate(rl.GetCameraUp(&camera), 0.5*rl.GetFrameTime())
			var view = rl.Vector3Subtract(camera.Position, camera.Target)
			view = rl.Vector3Transform(view, rotation)
			camera.Position = rl.Vector3Add(camera.Target, view)
		}

		rl.CameraMoveToTarget(&camera, -rl.GetMouseWheelMove())
		if rl.IsKeyPressed(rl.KeyKpSubtract) {
			rl.CameraMoveToTarget(&camera, 2.0)
		}
		if rl.IsKeyPressed(rl.KeyKpAdd) {
			rl.CameraMoveToTarget(&camera, -2.0)
		}

		const scaleFactor = 0.95
		if rl.IsKeyPressed(rl.KeyMinus) {
			inputWidth *= scaleFactor
			inputHeight *= scaleFactor
			padding *= scaleFactor
		}
		if rl.IsKeyPressed(rl.KeyEqual) {
			inputWidth /= scaleFactor
			inputHeight /= scaleFactor
			padding /= scaleFactor
		}

		rl.BeginDrawing()
		{
			rl.ClearBackground(rl.RayWhite)

			rl.BeginMode3D(camera)
			{
				rl.DrawGrid(32, 1)

				bodySizeV := InputsToVec3(bodySize)
				// rl.DrawCube(rl.NewVector3(0, bodySize.Y/2, 0), bodySize.X, bodySize.Y, bodySize.Z, rl.Red)
				rl.DrawCubeWires(rl.NewVector3(0, bodySizeV.Y/2, 0), bodySizeV.X, bodySizeV.Y, bodySizeV.Z, rl.Black)

				if false {
					for i, p := range body {
						rl.DrawCube(p, 0.1, 0.1, 0.1, rl.Blue)
						if showNumbers {
							rl.DrawBillboard(camera, numbers[i+1], rl.Vector3Add(p, rl.Vector3{Y: 0.2}), 0.2, rl.Black)
						}
					}
				} else {
					for _, i := range bodyOuterIndexes {
						p := body[i]
						rl.DrawCube(p, 0.1, 0.1, 0.1, rl.Blue)
						if showNumbers {
							rl.DrawBillboard(camera, numbers[i+1], rl.Vector3Add(p, rl.Vector3{Y: 0.2}), 0.2, rl.Black)
						}
					}
				}

				const thickness = 0.02
				rl.DrawCylinderEx(a0, aX, thickness, thickness, 8, rl.Red)
				rl.DrawCylinderEx(a0, aY, thickness, thickness, 8, rl.Green)
				rl.DrawCylinderEx(a0, aZ, thickness, thickness, 8, rl.Blue)
			}
			rl.EndMode3D()

			rl.DrawRectangleRec(topLeftUiRect, rl.RayWhite)
			rl.DrawRectangleLinesEx(topLeftUiRect, 1, rl.Gray)

			rl.DrawRectangleRec(bottomLeftUiRect, rl.RayWhite)
			rl.DrawRectangleLinesEx(bottomLeftUiRect, 1, rl.Gray)

			bodyUpdated := false

			// Sizes
			gui.Label(rl.NewRectangle(padding, padding, inputWidth/2, inputHeight), "Size")
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*(float32(i)+0.5), padding, inputWidth, inputHeight),
					&bodySize[i].Text, inputTextSize, bodySize[i].Edit,
				) {
					bodySize[i].ToggleEdit()
					v, err := strconv.ParseFloat(bodySize[i].Text, 32)
					if err != nil {
						slog.Error("Invalid size value", "err", err)
					} else {
						v = max(min(v, 1000), 0.01)
						bodySize[i].Value = v
						bodyUpdated = true
					}
					bodySize[i].UpdateText()
				}
			}

			// Splits
			gui.Label(rl.NewRectangle(padding, padding+padding+inputHeight, inputWidth/2, inputHeight), "Split")
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*(float32(i)+0.5), padding+inputHeight+padding, inputWidth, inputHeight),
					&bodySplit[i].Text, inputTextSize, bodySplit[i].Edit,
				) {
					bodySplit[i].ToggleEdit()
					v, err := strconv.ParseInt(bodySplit[i].Text, 10, 32)
					if err != nil {
						slog.Error("Invalid split value", "err", err, "value", v)
					} else {
						v = max(min(v, 100), 1)
						bodySplit[i].Value = int(v)
						bodyUpdated = true
					}
					bodySplit[i].UpdateText()
				}
			}

			// Show numbers
			showNumbers = gui.CheckBox(
				rl.NewRectangle(padding, padding+inputHeight*2+padding+padding, inputHeight, inputHeight),
				"Show Numbers", showNumbers,
			)

			// Young's modulus
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight), "Young's Modulus")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight),
				&yungaModule.Text, inputTextSize, yungaModule.Edit,
			) {
				yungaModule.ToggleEdit()
				v, err := strconv.ParseFloat(yungaModule.Text, 64)
				if err != nil {
					slog.Error("Invalid Young's modulus value", "err", err)
				} else {
					yungaModule.Value = max(min(v, 100000.0), 0.01) // TODO: Validate range
				}
				yungaModule.UpdateText()
			}

			// Poisson's ratio
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight), "Poisson's ratio")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight),
				&poissonRatio.Text, inputTextSize, poissonRatio.Edit,
			) {
				poissonRatio.ToggleEdit()
				v, err := strconv.ParseFloat(poissonRatio.Text, 64)
				if err != nil {
					slog.Error("Invalid Poisson's ratio value", "err", err)
				} else {
					poissonRatio.Value = max(min(v, 0.5), 0.0) // TODO: Validate range
				}
				poissonRatio.UpdateText()
			}

			// Pressure
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight), "Pressure")
			if gui.TextBox(
				rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight),
				&pressure.Text, inputTextSize, pressure.Edit,
			) {
				pressure.ToggleEdit()
				v, err := strconv.ParseFloat(pressure.Text, 64)
				if err != nil {
					slog.Error("Invalid pressure value", "err", err)
				} else {
					pressure.Value = max(min(v, 10000.0), 0.01) // TODO: Validate range
				}
				pressure.UpdateText()
			}

			if bodyUpdated {
				body, bodyOuterIndexes = buildBodyShape(InputsToVec3(bodySize), InputsToVec3(bodySplit))
			}

			// Run
			if gui.Button(
				rl.NewRectangle(float32(rl.GetScreenWidth())-padding-inputWidth, float32(rl.GetScreenHeight())-padding-inputHeight, inputWidth, inputHeight),
				"Run",
			) {
				slog.Info("Running...", "bodySize", InputsToVec3(bodySize), "bodySplits", InputsToVec3(bodySplit), "yungaModule", yungaModule, "poissonRatio", poissonRatio, "pressure", pressure)
				fem := &FEM{}
				fem.Solve(InputsToSlice3(bodySize), InputsToSlice3(bodySplit), yungaModule.Value, poissonRatio.Value, pressure.Value)
			}
		}
		rl.EndDrawing()
	}
}

func buildBodyShape(size rl.Vector3, splits rl.Vector3) ([]rl.Vector3, []int) {
	var body []rl.Vector3
	var bodyOuterIndexes []int
	splits = rl.Vector3Scale(splits, 2)
	offset := rl.NewVector3(size.X/2, 0, size.Z/2)

	for y := range int(splits.Y) + 1 {
		for x := int(splits.X); x >= 0; x-- {
			for z := int(splits.Z); z >= 0; z-- {
				if (x%2 != 0 && z%2 != 0) || (x%2 != 0 && y%2 != 0) || (y%2 != 0 && z%2 != 0) {
					continue
				}

				if x == 0 || y == 0 || z == 0 ||
					x == int(splits.X) || y == int(splits.Y) || z == int(splits.Z) {
					bodyOuterIndexes = append(bodyOuterIndexes, len(body))
				}

				body = append(body, rl.Vector3Subtract(rl.NewVector3(
					(float32(x)*size.X)/splits.X,
					(float32(y)*size.Y)/splits.Y,
					(float32(z)*size.Z)/splits.Z,
				), offset))
			}
		}
	}

	return body, bodyOuterIndexes
}
