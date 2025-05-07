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

	bodySize := rl.NewVector3(4, 4, 4)
	// bodySize := rl.NewVector3(4, 5, 3) // TODO: Reset
	bodySizesValue := [3]string{
		strconv.FormatFloat(float64(bodySize.X), 'f', 2, 64),
		strconv.FormatFloat(float64(bodySize.Y), 'f', 2, 64),
		strconv.FormatFloat(float64(bodySize.Z), 'f', 2, 64),
	}
	var bodySizesEdit [3]bool

	bodySplits := rl.NewVector3(2, 2, 2)
	// bodySplits := rl.NewVector3(4, 8, 3) // TODO: Reset
	bodySplitsValue := [3]string{
		strconv.FormatFloat(float64(bodySplits.X), 'f', 0, 64),
		strconv.FormatFloat(float64(bodySplits.Y), 'f', 0, 64),
		strconv.FormatFloat(float64(bodySplits.Z), 'f', 0, 64),
	}
	var bodySplitsEdit [3]bool

	yungaModule := 4.0
	yungaModuleValue := strconv.FormatFloat(yungaModule, 'f', 2, 64)
	yungaModuleEdit := false
	poissonRatio := 0.3
	poissonRatioValue := strconv.FormatFloat(poissonRatio, 'f', 2, 64)
	poissonRatioEdit := false
	pressure := 10.0
	pressureValue := strconv.FormatFloat(pressure, 'f', 2, 64)
	pressureEdit := false

	body, bodyOuterIndexes := buildBodyShape(bodySize, bodySplits)

	const (
		inputWidth    = 64
		inputHeight   = 24
		inputTextSize = 20
		padding       = 8
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

		rl.BeginDrawing()
		{
			rl.ClearBackground(rl.RayWhite)

			rl.BeginMode3D(camera)
			{
				rl.DrawGrid(32, 1)

				// rl.DrawCube(rl.NewVector3(0, bodySize.Y/2, 0), bodySize.X, bodySize.Y, bodySize.Z, rl.Red)
				rl.DrawCubeWires(rl.NewVector3(0, bodySize.Y/2, 0), bodySize.X, bodySize.Y, bodySize.Z, rl.Black)

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
					&bodySizesValue[i], inputTextSize, bodySizesEdit[i],
				) {
					bodySizesEdit[i] = !bodySizesEdit[i]

					v, err := strconv.ParseFloat(bodySizesValue[i], 32)
					if err != nil {
						slog.Error("Invalid size value", "err", err)
					} else {
						v = max(min(v, 1000), 0.01)
						switch i {
						case 0:
							bodySize.X = float32(v)
						case 1:
							bodySize.Y = float32(v)
						case 2:
							bodySize.Z = float32(v)
						}
						bodyUpdated = true
					}
					var sv string
					switch i {
					case 0:
						sv = strconv.FormatFloat(float64(bodySize.X), 'f', 2, 64)
					case 1:
						sv = strconv.FormatFloat(float64(bodySize.Y), 'f', 2, 64)
					case 2:
						sv = strconv.FormatFloat(float64(bodySize.Z), 'f', 2, 64)
					}
					bodySizesValue[i] = sv
				}
			}

			// Splits
			gui.Label(rl.NewRectangle(padding, padding+padding+inputHeight, inputWidth/2, inputHeight), "Split")
			for i := range 3 {
				if gui.TextBox(
					rl.NewRectangle(padding+(inputWidth+padding)*(float32(i)+0.5), padding+inputHeight+padding, inputWidth, inputHeight),
					&bodySplitsValue[i], inputTextSize, bodySplitsEdit[i],
				) {
					bodySplitsEdit[i] = !bodySplitsEdit[i]

					v, err := strconv.ParseInt(bodySplitsValue[i], 10, 32)
					if err != nil {
						slog.Error("Invalid split value", "err", err, "value", v)
					} else {
						v = max(min(v, 100), 1)
						switch i {
						case 0:
							bodySplits.X = float32(v)
						case 1:
							bodySplits.Y = float32(v)
						case 2:
							bodySplits.Z = float32(v)
						}
						bodyUpdated = true
					}
					var sv string
					switch i {
					case 0:
						sv = strconv.FormatInt(int64(bodySplits.X), 10)
					case 1:
						sv = strconv.FormatInt(int64(bodySplits.Y), 10)
					case 2:
						sv = strconv.FormatInt(int64(bodySplits.Z), 10)
					}
					bodySplitsValue[i] = sv
				}
			}

			// Show numbers
			showNumbers = gui.CheckBox(
				rl.NewRectangle(padding, padding+inputHeight*2+padding+padding, inputHeight, inputHeight),
				"Show Numbers", showNumbers,
			)

			// Young's modulus
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight), "Young's Modulus")
			if gui.TextBox(rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding, inputWidth, inputHeight), &yungaModuleValue, inputTextSize, yungaModuleEdit) {
				yungaModuleEdit = !yungaModuleEdit

				v, err := strconv.ParseFloat(yungaModuleValue, 64)
				if err != nil {
					slog.Error("Invalid Young's modulus value", "err", err)
				} else {
					yungaModule = max(min(v, 100000.0), 0.01) // TODO: Validate range
				}
				yungaModuleValue = strconv.FormatFloat(yungaModule, 'f', 2, 64)
			}

			// Poisson's ratio
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight), "Poisson's ratio")
			if gui.TextBox(rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+padding+inputHeight, inputWidth, inputHeight), &poissonRatioValue, inputTextSize, poissonRatioEdit) {
				poissonRatioEdit = !poissonRatioEdit

				v, err := strconv.ParseFloat(poissonRatioValue, 64)
				if err != nil {
					slog.Error("Invalid Poisson's ratio value", "err", err)
				} else {
					poissonRatio = max(min(v, 0.5), 0.0) // TODO: Validate range
				}
				poissonRatioValue = strconv.FormatFloat(poissonRatio, 'f', 2, 64)
			}

			// Pressure
			gui.Label(rl.NewRectangle(bottomLeftUiRect.X+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight), "Pressure")
			if gui.TextBox(rl.NewRectangle(bottomLeftUiRect.X+padding+inputWidth+padding, bottomLeftUiRect.Y+padding+(padding+inputHeight)*2, inputWidth, inputHeight), &pressureValue, inputTextSize, pressureEdit) {
				pressureEdit = !pressureEdit

				v, err := strconv.ParseFloat(pressureValue, 64)
				if err != nil {
					slog.Error("Invalid pressure value", "err", err)
				} else {
					pressure = max(min(v, 10000.0), 0.01) // TODO: Validate range
				}
				pressureValue = strconv.FormatFloat(pressure, 'f', 2, 64)
			}

			if bodyUpdated {
				body, bodyOuterIndexes = buildBodyShape(bodySize, bodySplits)
			}

			// Run
			if gui.Button(rl.NewRectangle(float32(rl.GetScreenWidth())-padding-inputWidth, float32(rl.GetScreenHeight())-padding-inputHeight, inputWidth, inputHeight), "Run") {
				slog.Info("Running...", "bodySize", bodySize, "bodySplits", bodySplits, "yungaModule", yungaModule, "poissonRatio", poissonRatio, "pressure", pressure)
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
